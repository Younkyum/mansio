package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/younkyumjin/lociterm/internal/tmux"
)

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

	var cols, rows uint16 = 120, 40

	attach, err := h.tmuxMgr.Attach(sessionID, cols, rows)
	if err != nil {
		log.Printf("tmux attach error for %s: %v", sessionID, err)
		conn.WriteJSON(ControlMessage{Type: "error", Message: err.Error()})
		return
	}
	sess := attach.Session
	// Pass the captured *Session back to Detach so a stale defer from an old
	// handler can't take down a newer handler's attach (compare-and-close).
	defer h.tmuxMgr.Detach(sessionID, sess)

	conn.WriteJSON(ControlMessage{Type: "attached", Shell: "tmux", Recreated: attach.Recreated})

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
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
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
					conn.WriteJSON(ControlMessage{Type: "pong"})
				}
			}
		}
	}()

	<-done
}
