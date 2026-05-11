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
const tmuxConfigContents = `# Managed by LociTerm — do not edit.
set -g mouse on
`

type Manager struct {
	mu          sync.RWMutex
	sessions    map[string]*Session
	shell       string
	configPath  string
	socketLabel string
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
	}
}

func (m *Manager) CreateSession(sessionID string, cols, rows uint16) error {
	name := sessionPrefix + sessionID

	home, _ := os.UserHomeDir()
	cmd := m.tmuxCmd("new-session", "-d", "-s", name,
		"-x", fmt.Sprintf("%d", cols), "-y", fmt.Sprintf("%d", rows),
		"-c", home,
		m.shell)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tmux new-session: %w: %s", err, string(out))
	}
	return nil
}

func (m *Manager) Attach(sessionID string, cols, rows uint16) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := sessionPrefix + sessionID

	if !m.tmuxSessionExists(name) {
		cmd := m.tmuxCmd("new-session", "-d", "-s", name,
			"-x", fmt.Sprintf("%d", cols), "-y", fmt.Sprintf("%d", rows),
			m.shell)
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("tmux new-session: %w: %s", err, string(out))
		}
	}

	sess, err := newSession(m.configPath, m.socketLabel, name, cols, rows)
	if err != nil {
		return nil, err
	}
	m.sessions[sessionID] = sess
	return sess, nil
}

func (m *Manager) Detach(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[sessionID]; ok {
		sess.Close()
		delete(m.sessions, sessionID)
	}
}

func (m *Manager) Resize(sessionID string, cols, rows uint16) error {
	name := sessionPrefix + sessionID

	cmd := m.tmuxCmd("resize-window", "-t", name,
		"-x", fmt.Sprintf("%d", cols), "-y", fmt.Sprintf("%d", rows))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tmux resize: %w: %s", err, string(out))
	}
	return nil
}

func (m *Manager) KillSession(sessionID string) error {
	m.Detach(sessionID)

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
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(line, sessionPrefix) {
			id := strings.TrimPrefix(line, sessionPrefix)
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
// LociTerm instance from the user's other tmux sessions and from other LociTerm instances
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

// Derives a tmux socket label like "lociterm-<8hex>" from the data-dir. Each instance with a
// distinct data-dir gets its own tmux server, so deletes in one instance can never reach
// sessions owned by another. Falls back to a fixed label if the path can't be resolved —
// still isolated from the user's default tmux server.
func deriveSocketLabel(dataDir string) string {
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		abs = dataDir
	}
	sum := sha256.Sum256([]byte(abs))
	return "lociterm-" + hex.EncodeToString(sum[:4])
}

// Writes tmux.conf if missing so users can edit it without us clobbering it on next launch.
func ensureTmuxConfig() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	dir := filepath.Join(base, "lociterm")
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
