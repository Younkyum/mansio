package tmux

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const sessionPrefix = "lt_"

// Without `mouse on`, tmux's alt-screen triggers xterm.js's wheel→arrow-key translation.
const tmuxConfigContents = `# Managed by Mansio — do not edit.
set -g mouse on
`

type Manager struct {
	mu          sync.RWMutex
	sessions    map[string]*Session
	shell       string
	configPath  string
	socketLabel string
	// lastSize tracks the last (cols, rows) we actually sent to tmux per session.
	// Skips redundant `tmux resize-window` shell-outs when client sends the same
	// dimensions repeatedly (common with ResizeObserver storms before the
	// frontend-side dedupe lands).
	lastSize map[string]struct{ cols, rows uint16 }
	// resizeShellOut is the function used to invoke `tmux resize-window`.
	// Tests override this to count invocations and verify dedupe behaviour
	// without requiring a live tmux server. nil means "use the real tmux command".
	resizeShellOut func(name string, cols, rows uint16) error
}

func NewManager(dataDir string) *Manager {
	shell := detectShell()
	configPath, err := ensureTmuxConfig()
	if err != nil {
		// A missing config means tmux falls back to user defaults; log and continue.
		fmt.Fprintf(os.Stderr, "tmux: config setup failed: %v\n", err)
	}
	return &Manager{
		sessions:    make(map[string]*Session),
		shell:       shell,
		configPath:  configPath,
		socketLabel: deriveSocketLabel(dataDir),
		lastSize:    make(map[string]struct{ cols, rows uint16 }),
	}
}

func (m *Manager) CreateSession(sessionID string, cols, rows uint16) error {
	name := sessionPrefix + sessionID

	cmd := m.newSessionCmd(name, cols, rows)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tmux new-session: %w: %s", err, string(out))
	}
	return nil
}

// AttachResult bundles the new attach handle with metadata the WS handler
// needs to surface to the frontend.
type AttachResult struct {
	Session *Session
	// Recreated is true when Attach had to spawn a fresh tmux session because
	// the prior one was gone (shell exited, host restarted, kill-session ran).
	// The caller turns this into a user-visible "session was lost" banner so
	// silent re-creation stops masquerading as a clean reattach.
	Recreated bool
}

// Attach hands the caller an active tmux attach client for sessionID, creating
// the underlying tmux session if missing. Returns AttachResult.Recreated=true
// when no prior tmux session existed, so the caller can flag the loss to the
// user instead of dropping them into a silent fresh shell.
//
// Concurrency contract: callers MUST keep the returned *Session reference and
// pass it back to Detach so a stale reference (e.g. an old WS handler still
// draining) cannot clobber a newer attach that has taken its slot in the
// internal map. Without this, the prior bug was that Detach looked up by ID
// only and would Close() the currently-live attach.
func (m *Manager) Attach(sessionID string, cols, rows uint16) (*AttachResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := sessionPrefix + sessionID

	recreated := false
	if !m.tmuxSessionExists(name) {
		cmd := m.newSessionCmd(name, cols, rows)
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("tmux new-session: %w: %s", err, string(out))
		}
		recreated = true
	}

	sess, err := newSession(m.configPath, m.socketLabel, name, cols, rows)
	if err != nil {
		return nil, err
	}

	// Defensive: if a prior *Session is still in the map for this ID (rare —
	// usually the old handler's Detach has cleared it first, but a hung
	// handler can leave it behind), close it before overwriting so its
	// resources aren't leaked. The compare-and-close guard in Detach prevents
	// the *next* old-handler return from clobbering this new attach.
	if prior, ok := m.sessions[sessionID]; ok {
		prior.Close()
	}
	m.sessions[sessionID] = sess
	return &AttachResult{Session: sess, Recreated: recreated}, nil
}

// Detach tears down the caller's attach client and removes it from the map,
// but ONLY when the map slot still points at the caller's session. This
// compare-and-close pattern prevents an orphaned old handler (whose deferred
// cleanup runs after a newer handler has already taken its slot) from killing
// the currently-active attach. If sess is nil, Detach is a no-op (used for
// admin paths that already cleaned up).
func (m *Manager) Detach(sessionID string, sess *Session) {
	if sess == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if current, ok := m.sessions[sessionID]; ok && current == sess {
		delete(m.sessions, sessionID)
	}
	// Closing outside the map check is safe: Session.Close is idempotent and
	// the caller's reference is unique. Doing it unconditionally guarantees
	// the attach client process is reaped even if the slot was already taken
	// by a newer handler (which owns its own session and won't be affected).
	sess.Close()
}

func (m *Manager) newSessionCmd(name string, cols, rows uint16) *exec.Cmd {
	args := []string{
		"new-session", "-d", "-s", name,
		"-x", fmt.Sprintf("%d", cols), "-y", fmt.Sprintf("%d", rows),
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		args = append(args, "-c", home)
	}
	args = append(args, m.shell)
	return m.tmuxCmd(args...)
}

func (m *Manager) Resize(sessionID string, cols, rows uint16) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if last, ok := m.lastSize[sessionID]; ok && last.cols == cols && last.rows == rows {
		return nil
	}

	name := sessionPrefix + sessionID
	if m.resizeShellOut != nil {
		if err := m.resizeShellOut(name, cols, rows); err != nil {
			return err
		}
	} else {
		cmd := m.tmuxCmd("resize-window", "-t", name,
			"-x", fmt.Sprintf("%d", cols), "-y", fmt.Sprintf("%d", rows))
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("tmux resize: %w: %s", err, string(out))
		}
	}

	m.lastSize[sessionID] = struct{ cols, rows uint16 }{cols, rows}
	return nil
}

// KillSession is the admin path used when a user deletes a tab/workspace.
// Unlike Detach, it tears down whatever attach is in the map (the user wants
// this gone regardless of which handler owns it) and then runs the tmux
// kill-session to drop the server-side session as well.
func (m *Manager) KillSession(sessionID string) error {
	m.mu.Lock()
	if sess, ok := m.sessions[sessionID]; ok {
		sess.Close()
		delete(m.sessions, sessionID)
	}
	delete(m.lastSize, sessionID)
	m.mu.Unlock()

	name := sessionPrefix + sessionID
	cmd := m.tmuxCmd("kill-session", "-t", name)
	cmd.Run()
	return nil
}

// GetCwd returns the current working directory of the session's primary pane.
// Returns "" if tmux is unavailable or the session doesn't exist (graceful: the
// sidebar treats empty CWD as "don't display"). The home prefix is shortened
// to "~" for display friendliness.
func (m *Manager) GetCwd(sessionID string) string {
	name := sessionPrefix + sessionID
	cmd := m.tmuxCmd("display-message", "-t", name, "-p", "#{pane_current_path}")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	cwd := strings.TrimSpace(string(out))
	if cwd == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		if cwd == home {
			return "~"
		}
		if strings.HasPrefix(cwd, home+"/") {
			return "~" + cwd[len(home):]
		}
	}
	return cwd
}

func (m *Manager) ListTmuxSessions() ([]string, error) {
	cmd := m.tmuxCmd("list-sessions", "-F", "#{session_name}")
	out, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "no server running") ||
			strings.Contains(string(out), "no server running") {
			return nil, nil
		}
		return nil, nil
	}

	var sessions []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if id, ok := strings.CutPrefix(line, sessionPrefix); ok {
			sessions = append(sessions, id)
		}
	}
	return sessions, nil
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, sess := range m.sessions {
		sess.Close()
		delete(m.sessions, id)
	}
}

func (m *Manager) tmuxSessionExists(name string) bool {
	cmd := m.tmuxCmd("has-session", "-t", name)
	return cmd.Run() == nil
}

// `-L` selects which tmux server we talk to and must be on every command; it isolates this
// Mansio instance from the user's other tmux sessions and from other Mansio instances
// pointing at a different data-dir. `-f` is honoured only on the command that starts the
// server; subsequent invocations ignore it.
func (m *Manager) tmuxCmd(args ...string) *exec.Cmd {
	prefix := []string{"-L", m.socketLabel}
	if m.configPath != "" {
		prefix = append(prefix, "-f", m.configPath)
	}
	return exec.Command("tmux", append(prefix, args...)...)
}

func detectShell() string {
	for _, sh := range []string{"/bin/zsh", "/bin/bash", "/bin/sh"} {
		if _, err := exec.LookPath(sh); err == nil {
			return sh
		}
	}
	return "/bin/sh"
}

// Derives a tmux socket label like "mansio-<8hex>" from the data-dir. Each instance with a
// distinct data-dir gets its own tmux server, so deletes in one instance can never reach
// sessions owned by another. Falls back to a fixed label if the path can't be resolved —
// still isolated from the user's default tmux server.
func deriveSocketLabel(dataDir string) string {
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		abs = dataDir
	}
	sum := sha256.Sum256([]byte(abs))
	return "mansio-" + hex.EncodeToString(sum[:4])
}

// Writes tmux.conf if missing so users can edit it without us clobbering it on next launch.
func ensureTmuxConfig() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	dir := filepath.Join(base, "mansio")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	path := filepath.Join(dir, "tmux.conf")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := os.WriteFile(path, []byte(tmuxConfigContents), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}
