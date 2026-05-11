package tmux

import (
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
	if !strings.HasPrefix(got, "lociterm-") {
		t.Fatalf("label %q missing lociterm- prefix; would collide with the user's default tmux server", got)
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
// test runs against its own isolated tmux server (-L lociterm-<hash>). Race
// tests can no longer collide with parallel runs or stray sessions on the
// user's default tmux server.
func newTestManager(t *testing.T) *Manager {
	t.Helper()
	return NewManager(t.TempDir())
}

func killAllLtSessions(m *Manager) {
	out, _ := m.tmuxCmd("list-sessions", "-F", "#{session_name}").Output()
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
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
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Detach(id, first.Session)
		}()
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
