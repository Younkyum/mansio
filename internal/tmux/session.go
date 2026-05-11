package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

type Session struct {
	mu     sync.Mutex
	name   string
	cmd    *exec.Cmd
	ptmx   *os.File
	closed bool
}

func newSession(configPath, socketLabel, tmuxName string, cols, rows uint16) (*Session, error) {
	args := []string{"-L", socketLabel}
	if configPath != "" {
		args = append(args, "-f", configPath)
	}
	args = append(args, "attach-session", "-t", tmuxName)
	cmd := exec.Command("tmux", args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: cols,
		Rows: rows,
	})
	if err != nil {
		return nil, fmt.Errorf("pty start tmux attach: %w", err)
	}

	return &Session{
		name: tmuxName,
		cmd:  cmd,
		ptmx: ptmx,
	}, nil
}

func (s *Session) Read(p []byte) (int, error) {
	return s.ptmx.Read(p)
}

func (s *Session) Write(p []byte) (int, error) {
	return s.ptmx.Write(p)
}

func (s *Session) Resize(cols, rows uint16) error {
	return pty.Setsize(s.ptmx, &pty.Winsize{
		Cols: cols,
		Rows: rows,
	})
}

func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.closed = true

	s.ptmx.Close()
	if s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}
}

// ClosePTMX closes only the pty side, which unblocks any goroutine stuck in
// Read/Write. Unlike Close, it does NOT kill the tmux attach client — that
// happens later when Close runs from the manager's Detach. The split exists
// because the WS handler needs to break its Read/Write loops promptly when
// either side of the pipe dies, but Close (which also kills the process and
// Waits) is owned by the manager so the compare-and-close identity check is
// honoured first.
func (s *Session) ClosePTMX() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	// Note: we intentionally do not set s.closed here. Close still needs to
	// run to kill the process and reap it. ptmx.Close is idempotent.
	_ = s.ptmx.Close()
}

func (s *Session) Done() <-chan error {
	ch := make(chan error, 1)
	go func() {
		ch <- s.cmd.Wait()
	}()
	return ch
}
