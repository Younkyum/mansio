package ws

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/younkyumjin/mansio/internal/tmux"
)

const writeTimeout = 30 * time.Second

var setTCPNoDelay = func(conn *net.TCPConn, noDelay bool) error {
	return conn.SetNoDelay(noDelay)
}

type wsWriter struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (w *wsWriter) writeBinary(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return w.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (w *wsWriter) writeJSON(v any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return w.conn.WriteJSON(v)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  32 * 1024,
	WriteBufferSize: 32 * 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Handler struct {
	tmuxMgr *tmux.Manager
}

func NewHandler(tmuxMgr *tmux.Manager) *Handler {
	return &Handler{tmuxMgr: tmuxMgr}
}

func (h *Handler) HandleTerminal(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	if sessionID == "" {
		http.Error(w, "session id required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()
	// Disable Nagle on the underlying TCP conn — CF tunnel adds another hop
	// via cloudflared, so explicit assertion is safer than relying on Go default.
	if tcp, ok := conn.NetConn().(*net.TCPConn); ok {
		if err := setTCPNoDelay(tcp, true); err != nil {
			log.Printf("ws: SetNoDelay failed for %s: %v", sessionID, err)
		}
	}
	writer := &wsWriter{conn: conn}

	var cols, rows uint16 = 120, 40

	attach, err := h.tmuxMgr.Attach(sessionID, cols, rows)
	if err != nil {
		log.Printf("tmux attach error for %s: %v", sessionID, err)
		_ = writer.writeJSON(ControlMessage{Type: "error", Message: err.Error()})
		return
	}
	sess := attach.Session
	// Pass the captured *Session back to Detach so a stale defer from an old
	// handler can't take down a newer handler's attach (compare-and-close).
	defer h.tmuxMgr.Detach(sessionID, sess)

	if err := writer.writeJSON(ControlMessage{Type: "attached", Shell: "tmux", Recreated: attach.Recreated}); err != nil {
		return
	}

	// done is closed by whichever side dies first. sync.Once makes the close
	// idempotent so both goroutines can safely call shutdown(). When one side
	// returns, shutdown() also closes the underlying conn and ptmx so the
	// *other* goroutine is forced out of its blocking read instead of hanging
	// until tmux happens to emit output.
	done := make(chan struct{})
	var once sync.Once
	shutdown := func() {
		once.Do(func() {
			close(done)
			// Closing conn unblocks the WS-read goroutine.
			conn.Close()
			// Closing ptmx unblocks the PTY-read goroutine. Without this,
			// dead-WS + idle-tmux leaves sess.Read pinned forever.
			sess.ClosePTMX()
		})
	}

	// PTY stdout -> WebSocket (binary frames)
	go func() {
		defer shutdown()
		buf := make([]byte, 32*1024)
		for {
			n, err := sess.Read(buf)
			if err != nil {
				return
			}
			if err := writer.writeBinary(buf[:n]); err != nil {
				return
			}
		}
	}()

	// WebSocket -> PTY stdin or control handler
	go func() {
		defer shutdown()
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				return
			}

			switch msgType {
			case websocket.BinaryMessage:
				sess.Write(data)
			case websocket.TextMessage:
				var ctrl ControlMessage
				if err := json.Unmarshal(data, &ctrl); err != nil {
					continue
				}
				switch ctrl.Type {
				case "resize":
					if ctrl.Cols > 0 && ctrl.Rows > 0 {
						sess.Resize(ctrl.Cols, ctrl.Rows)
						h.tmuxMgr.Resize(sessionID, ctrl.Cols, ctrl.Rows)
					}
				case "ping":
					if err := writer.writeJSON(ControlMessage{Type: "pong"}); err != nil {
						return
					}
				}
			}
		}
	}()

	<-done
}
