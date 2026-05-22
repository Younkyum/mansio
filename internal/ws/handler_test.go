package ws

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/younkyumjin/mansio/internal/tmux"
)

func TestSetNoDelayCalled(t *testing.T) {
	const sessionID = "set-nodelay-called"
	mgr := newTestManager(t, sessionID)

	originalSetTCPNoDelay := setTCPNoDelay
	var calls atomic.Int32
	var sawDisabled atomic.Bool
	setTCPNoDelay = func(conn *net.TCPConn, noDelay bool) error {
		calls.Add(1)
		if !noDelay {
			sawDisabled.Store(true)
		}
		return originalSetTCPNoDelay(conn, noDelay)
	}
	t.Cleanup(func() { setTCPNoDelay = originalSetTCPNoDelay })

	listener := listenLocal(t)
	baseURL := startTerminalServer(t, listener, mgr)
	conn := dialTerminal(t, fmt.Sprintf("%s/%s", baseURL, sessionID))

	readControl(t, conn, "attached", 5*time.Second)

	if sawDisabled.Load() {
		t.Fatalf("SetNoDelay called with false")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("SetNoDelay calls = %d, want 1", got)
	}
}

func TestWriteDeadline(t *testing.T) {
	yesPath, err := exec.LookPath("yes")
	if err != nil {
		t.Skip("yes not installed on PATH; skipping write-deadline integration test")
	}

	const sessionID = "write-deadline"
	mgr := newTestManager(t, sessionID)
	listener := newBlockingWriteListener(t)
	baseURL := startTerminalServer(t, listener, mgr)
	conn := dialTerminal(t, fmt.Sprintf("%s/%s", baseURL, sessionID))

	readControl(t, conn, "attached", 5*time.Second)

	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set client write deadline: %v", err)
	}
	command := fmt.Sprintf("%q mansio-write-deadline\n", yesPath)
	if err := conn.WriteMessage(websocket.BinaryMessage, []byte(command)); err != nil {
		t.Fatalf("start output command: %v", err)
	}
	listener.blockWrites.Store(true)

	// Simulate a client that stops draining the socket: after the command starts
	// producing PTY output, the server-side net.Conn blocks writes until
	// wsWriter's 30 s SetWriteDeadline expires.
	var blockedAt time.Time
	select {
	case <-listener.blocked:
		blockedAt = time.Now()
	case <-time.After(5 * time.Second):
		t.Fatalf("server websocket write did not block")
	}

	closeBy := blockedAt.Add(60 * time.Second)
	if err := conn.SetReadDeadline(closeBy); err != nil {
		t.Fatalf("set client read deadline: %v", err)
	}
	for {
		_, _, err := conn.ReadMessage()
		if err == nil {
			continue
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			t.Fatalf("websocket did not close within 60s after the client stopped reading")
		}
		if elapsed := time.Since(blockedAt); elapsed < writeTimeout-2*time.Second {
			t.Fatalf("websocket closed after %s, before the 30s write deadline", elapsed)
		}
		return
	}
}

func TestSetNoDelayFallback(t *testing.T) {
	const sessionID = "set-nodelay-fallback"
	mgr := newTestManager(t, sessionID)

	originalSetTCPNoDelay := setTCPNoDelay
	var calls atomic.Int32
	setTCPNoDelay = func(conn *net.TCPConn, noDelay bool) error {
		calls.Add(1)
		return originalSetTCPNoDelay(conn, noDelay)
	}
	t.Cleanup(func() { setTCPNoDelay = originalSetTCPNoDelay })

	listener := nonTCPListener{Listener: listenLocal(t)}
	baseURL := startTerminalServer(t, listener, mgr)
	conn := dialTerminal(t, fmt.Sprintf("%s/%s", baseURL, sessionID))

	readControl(t, conn, "attached", 5*time.Second)
	if got := calls.Load(); got != 0 {
		t.Fatalf("SetNoDelay calls over non-TCP conn = %d, want 0", got)
	}

	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set client write deadline: %v", err)
	}
	if err := conn.WriteJSON(ControlMessage{Type: "ping"}); err != nil {
		t.Fatalf("send ping: %v", err)
	}
	readControl(t, conn, "pong", 5*time.Second)
}

type nonTCPConn struct {
	net.Conn
}

type nonTCPListener struct {
	net.Listener
}

func (l nonTCPListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return nonTCPConn{Conn: conn}, nil
}

type blockingWriteListener struct {
	net.Listener
	blockWrites atomic.Bool
	blocked     chan struct{}
	blockedOnce sync.Once
}

func newBlockingWriteListener(t *testing.T) *blockingWriteListener {
	t.Helper()
	return &blockingWriteListener{
		Listener: listenLocal(t),
		blocked:  make(chan struct{}),
	}
}

func (l *blockingWriteListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &blockingWriteConn{
		Conn:        conn,
		blockWrites: &l.blockWrites,
		blocked:     l.blocked,
		blockedOnce: &l.blockedOnce,
		closed:      make(chan struct{}),
	}, nil
}

type blockingWriteConn struct {
	net.Conn
	blockWrites  *atomic.Bool
	blocked      chan struct{}
	blockedOnce  *sync.Once
	closed       chan struct{}
	closeOnce    sync.Once
	writeTimeout atomic.Int64
}

func (c *blockingWriteConn) Write(p []byte) (int, error) {
	if !c.blockWrites.Load() {
		return c.Conn.Write(p)
	}
	c.blockedOnce.Do(func() { close(c.blocked) })

	for {
		deadlineUnix := c.writeTimeout.Load()
		if deadlineUnix != 0 {
			deadline := time.Unix(0, deadlineUnix)
			if wait := time.Until(deadline); wait > 0 {
				timer := time.NewTimer(wait)
				select {
				case <-timer.C:
					return 0, simulatedWriteTimeout{}
				case <-c.closed:
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					return 0, net.ErrClosed
				}
			}
			return 0, simulatedWriteTimeout{}
		}

		select {
		case <-c.closed:
			return 0, net.ErrClosed
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (c *blockingWriteConn) SetDeadline(t time.Time) error {
	c.storeWriteDeadline(t)
	return c.Conn.SetDeadline(t)
}

func (c *blockingWriteConn) SetWriteDeadline(t time.Time) error {
	c.storeWriteDeadline(t)
	return c.Conn.SetWriteDeadline(t)
}

func (c *blockingWriteConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

func (c *blockingWriteConn) storeWriteDeadline(t time.Time) {
	if t.IsZero() {
		c.writeTimeout.Store(0)
		return
	}
	c.writeTimeout.Store(t.UnixNano())
}

type simulatedWriteTimeout struct{}

func (simulatedWriteTimeout) Error() string { return "simulated websocket write timeout" }

func (simulatedWriteTimeout) Timeout() bool { return true }

func (simulatedWriteTimeout) Temporary() bool { return true }

func requireTmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed on PATH; skipping websocket integration test")
	}
}

func newTestManager(t *testing.T, sessionID string) *tmux.Manager {
	t.Helper()
	requireTmux(t)
	mgr := tmux.NewManager(t.TempDir())
	t.Cleanup(func() {
		_ = mgr.KillSession(sessionID)
		mgr.Shutdown()
	})
	return mgr
}

func listenLocal(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return listener
}

func startTerminalServer(t *testing.T, listener net.Listener, mgr *tmux.Manager) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws/{sessionId}", NewHandler(mgr).HandleTerminal)

	server := &http.Server{Handler: mux}
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.Serve(listener) }()

	t.Cleanup(func() {
		_ = server.Close()
		select {
		case err := <-serverErr:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				t.Errorf("server: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("server did not stop")
		}
	})

	return "ws://" + listener.Addr().String() + "/ws"
}

func dialTerminal(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readControl(t *testing.T, conn *websocket.Conn, wantType string, timeout time.Duration) ControlMessage {
	t.Helper()
	deadline := time.Now().Add(timeout)
	if err := conn.SetReadDeadline(deadline); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				t.Fatalf("timed out waiting for %q control message", wantType)
			}
			t.Fatalf("read %q control message: %v", wantType, err)
		}
		if msgType != websocket.TextMessage {
			continue
		}

		var ctrl ControlMessage
		if err := json.Unmarshal(data, &ctrl); err != nil {
			t.Fatalf("decode control message %q: %v", data, err)
		}
		if ctrl.Type == wantType {
			return ctrl
		}
	}
}
