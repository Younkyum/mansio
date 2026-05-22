package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestDeriveSocketLabel_DistinctDataDirsGetDistinctLabels(t *testing.T) {
	a := deriveSocketLabel("/tmp/prod-data")
	b := deriveSocketLabel("/tmp/dev-data")
	if a == b {
		t.Fatalf("expected distinct labels for distinct data dirs, both = %q", a)
	}
}

func TestDeriveSocketLabel_StableAcrossEquivalentPaths(t *testing.T) {
	abs, err := filepath.Abs("./data")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if deriveSocketLabel("./data") != deriveSocketLabel(abs) {
		t.Fatalf("relative and absolute forms of the same path must hash identically")
	}
}

func TestDeriveSocketLabel_HasLociPrefix(t *testing.T) {
	got := deriveSocketLabel("/tmp/anything")
	if !strings.HasPrefix(got, "mansio-") {
		t.Fatalf("label %q missing mansio- prefix; would collide with the user's default tmux server", got)
	}
}

// requireTmux skips the test if tmux is not available on PATH. The race
// reproducer needs a real attach client process to exercise the bug.
func requireTmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed on PATH; skipping integration test")
	}
}

// newTestManager spins up a Manager pointed at a per-test temp dir so each
// test runs against its own isolated tmux server (-L mansio-<hash>). Race
// tests can no longer collide with parallel runs or stray sessions on the
// user's default tmux server.
func newTestManager(t *testing.T) *Manager {
	t.Helper()
	return NewManager(t.TempDir())
}

func killAllLtSessions(m *Manager) {
	out, _ := m.tmuxCmd("list-sessions", "-F", "#{session_name}").Output()
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(line, sessionPrefix) {
			m.tmuxCmd("kill-session", "-t", line).Run()
		}
	}
}

// Regression for the "tmux exits during work" bug: an orphaned old handler's
// deferred Detach used to look up the session by ID only, so when a newer
// handler had taken the map slot, the stale Detach would Close() the live
// attach client and the user's working terminal would die. The fix is the
// compare-and-close pattern: Detach now takes the caller's *Session and only
// clears the slot when it still belongs to that caller.
func TestDetach_DoesNotClobberNewerSessionOnStaleDetach(t *testing.T) {
	requireTmux(t)
	m := newTestManager(t)
	t.Cleanup(func() { killAllLtSessions(m) })

	const id = "race-id-1"
	first, err := m.Attach(id, 80, 24)
	if err != nil {
		t.Fatalf("first attach: %v", err)
	}
	second, err := m.Attach(id, 80, 24)
	if err != nil {
		t.Fatalf("second attach: %v", err)
	}
	if first.Session == second.Session {
		t.Fatalf("expected distinct *Session objects across two Attach calls")
	}

	// Simulate the orphaned old handler firing its deferred Detach AFTER the
	// new attach has taken the slot. With the bug, this would Close the
	// second session's attach client; with the fix, the second session
	// survives because m.sessions[id] != first.Session.
	m.Detach(id, first.Session)

	m.mu.RLock()
	current, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		t.Fatalf("second session was evicted from the manager map by a stale Detach")
	}
	if current != second.Session {
		t.Fatalf("map slot points at unexpected *Session after stale Detach")
	}

	// And the live attach client's PTY should still be usable. A trivial way
	// to prove "not killed" is to check the cmd has not been reaped.
	if second.Session.cmd.ProcessState != nil {
		t.Fatalf("live attach client was killed by the stale Detach (ProcessState=%v)", second.Session.cmd.ProcessState)
	}

	// Real Detach for the active session — should drop the slot.
	m.Detach(id, second.Session)
	m.mu.RLock()
	_, stillThere := m.sessions[id]
	m.mu.RUnlock()
	if stillThere {
		t.Fatalf("matching Detach failed to evict the active session")
	}
}

// Attach should also defensively close any pre-existing *Session that is
// still parked in the map for the same ID (covers the case where two attach
// requests race and the second arrives before the first's Detach has fired).
func TestAttach_ClosesPriorOrphanInMap(t *testing.T) {
	requireTmux(t)
	m := newTestManager(t)
	t.Cleanup(func() { killAllLtSessions(m) })

	const id = "orphan-id-2"
	first, err := m.Attach(id, 80, 24)
	if err != nil {
		t.Fatalf("first attach: %v", err)
	}

	// Don't Detach first — simulate the leaked-orphan case.
	second, err := m.Attach(id, 80, 24)
	if err != nil {
		t.Fatalf("second attach: %v", err)
	}

	// The map slot should now be the second session, and the first should
	// have been closed by Attach's defensive cleanup.
	m.mu.RLock()
	current := m.sessions[id]
	m.mu.RUnlock()
	if current != second.Session {
		t.Fatalf("map slot was not updated to the second session")
	}
	first.Session.mu.Lock()
	firstClosed := first.Session.closed
	first.Session.mu.Unlock()
	if !firstClosed {
		t.Fatalf("first session was not closed by Attach's defensive cleanup")
	}
}

func TestDetach_NilSessionIsNoOp(t *testing.T) {
	m := newTestManager(t)
	// Should not panic and should not touch the map.
	m.Detach("anything", nil)
	if len(m.sessions) != 0 {
		t.Fatalf("Detach(nil) altered the sessions map")
	}
}

// Recreated must be true when Attach has to spawn a fresh tmux session, and
// false when it reattaches to an existing one. This is what powers the
// "previous session was lost" banner in the frontend.
func TestAttach_RecreatedFlag(t *testing.T) {
	requireTmux(t)
	m := newTestManager(t)
	t.Cleanup(func() { killAllLtSessions(m) })

	const id = "recreate-id-3"
	first, err := m.Attach(id, 80, 24)
	if err != nil {
		t.Fatalf("first attach: %v", err)
	}
	if !first.Recreated {
		t.Fatalf("first Attach should report Recreated=true (no prior tmux session existed)")
	}

	// Detach + ensure the tmux server-side session is still alive.
	m.Detach(id, first.Session)

	second, err := m.Attach(id, 80, 24)
	if err != nil {
		t.Fatalf("second attach: %v", err)
	}
	if second.Recreated {
		t.Fatalf("second Attach should report Recreated=false (reattached to existing session)")
	}
	m.Detach(id, second.Session)
}

func TestAttach_StartsFreshSessionInHomeDirectory(t *testing.T) {
	requireTmux(t)
	m := newTestManager(t)
	t.Cleanup(func() { killAllLtSessions(m) })

	const id = "home-cwd-id"
	attach, err := m.Attach(id, 80, 24)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	defer m.Detach(id, attach.Session)

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("home dir: %v", err)
	}
	name := sessionPrefix + id
	out, err := m.tmuxCmd("display-message", "-t", name, "-p", "#{pane_current_path}").Output()
	if err != nil {
		t.Fatalf("display cwd: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != home {
		t.Fatalf("fresh session cwd = %q, want home %q", got, home)
	}
}

func TestNewSessionCmdOmitsCwdWhenHomeUnavailable(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("home", "")

	m := newTestManager(t)
	cmd := m.newSessionCmd("lt_no-home", 80, 24)
	args := fmt.Sprint(cmd.Args)
	if strings.Contains(args, " -c ") {
		t.Fatalf("newSessionCmd should omit -c when no home dir is available, args=%v", cmd.Args)
	}
}

// TestResizeDedupe ensures repeated Resize calls with identical dimensions
// shell out to `tmux resize-window` exactly once. ResizeObserver storms in
// the browser otherwise translate into hundreds of redundant tmux exec()s.
func TestResizeDedupe(t *testing.T) {
	m := newTestManager(t)

	const id = "dedupe-id"
	var calls int
	m.resizeShellOut = func(name string, cols, rows uint16) error {
		if name != sessionPrefix+id {
			t.Errorf("unexpected tmux target name %q", name)
		}
		calls++
		return nil
	}

	for range 100 {
		if err := m.Resize(id, 80, 24); err != nil {
			t.Fatalf("Resize: %v", err)
		}
	}

	if calls != 1 {
		t.Fatalf("expected 1 shell-out for 100 identical Resize calls, got %d", calls)
	}

	m.mu.RLock()
	cached, ok := m.lastSize[id]
	m.mu.RUnlock()
	if !ok || cached.cols != 80 || cached.rows != 24 {
		t.Fatalf("lastSize cache state wrong: ok=%v cached=%+v", ok, cached)
	}
}

// TestResizeDifferentDimensions confirms only identical dimensions are
// deduped — every distinct (cols, rows) pair must reach tmux.
func TestResizeDifferentDimensions(t *testing.T) {
	m := newTestManager(t)

	const id = "diff-id"
	var calls int
	m.resizeShellOut = func(name string, cols, rows uint16) error {
		calls++
		return nil
	}

	dims := []struct{ cols, rows uint16 }{
		{80, 24}, {100, 30}, {120, 40}, {80, 24}, {120, 40},
	}
	for _, d := range dims {
		if err := m.Resize(id, d.cols, d.rows); err != nil {
			t.Fatalf("Resize(%d,%d): %v", d.cols, d.rows, err)
		}
	}

	// 5 inputs: {80,24},{100,30},{120,40},{80,24},{120,40}
	// → shell-outs at indices 0,1,2,3,4 (index 3 changes from {120,40} back
	// to {80,24}, index 4 changes from {80,24} back to {120,40}; both are
	// distinct from the immediately preceding cached value).
	if calls != 5 {
		t.Fatalf("expected 5 shell-outs for 5 distinct-from-previous Resize calls, got %d", calls)
	}
}

// TestResizeAfterKillSession verifies KillSession clears the dedupe cache so
// that re-creating a session with the same ID resizes fresh tmux state and
// is not silently skipped by a stale cache entry.
func TestResizeAfterKillSession(t *testing.T) {
	m := newTestManager(t)

	const id = "kill-cache-id"
	var calls int
	m.resizeShellOut = func(name string, cols, rows uint16) error {
		calls++
		return nil
	}

	if err := m.Resize(id, 80, 24); err != nil {
		t.Fatalf("first Resize: %v", err)
	}
	if calls != 1 {
		t.Fatalf("first Resize should shell out once, got %d", calls)
	}

	// KillSession runs `tmux kill-session` which fails harmlessly when no
	// server is running; we only care that it clears m.lastSize.
	if err := m.KillSession(id); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	m.mu.RLock()
	_, stillCached := m.lastSize[id]
	m.mu.RUnlock()
	if stillCached {
		t.Fatalf("KillSession failed to clear lastSize entry for %q", id)
	}

	if err := m.Resize(id, 80, 24); err != nil {
		t.Fatalf("post-kill Resize: %v", err)
	}
	if calls != 2 {
		t.Fatalf("post-kill Resize should shell out again, got total calls = %d", calls)
	}
}

// Ensures Detach is safe to call concurrently from many stale handlers
// against the same ID — none of them must clobber the active session.
func TestDetach_ConcurrentStaleCallsDoNotClobberActive(t *testing.T) {
	requireTmux(t)
	m := newTestManager(t)
	t.Cleanup(func() { killAllLtSessions(m) })

	const id = "concurrent-id-4"
	first, err := m.Attach(id, 80, 24)
	if err != nil {
		t.Fatalf("first attach: %v", err)
	}
	second, err := m.Attach(id, 80, 24)
	if err != nil {
		t.Fatalf("second attach: %v", err)
	}

	// Many stale Detach calls referencing the first (already orphaned) session.
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			m.Detach(id, first.Session)
		})
	}
	wg.Wait()

	m.mu.RLock()
	current := m.sessions[id]
	m.mu.RUnlock()
	if current != second.Session {
		t.Fatalf("concurrent stale Detach calls clobbered the active session")
	}
	m.Detach(id, second.Session)
}
