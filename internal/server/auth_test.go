package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthManager(t *testing.T) {
	am := newAuthManager()

	t.Run("create and validate session", func(t *testing.T) {
		token := am.createSession()
		if token == "" {
			t.Fatal("token should not be empty")
		}
		if !am.validateSession(token) {
			t.Error("newly created session should be valid")
		}
	})

	t.Run("invalid token rejected", func(t *testing.T) {
		if am.validateSession("nonexistent-token") {
			t.Error("nonexistent token should be invalid")
		}
	})

	t.Run("delete session", func(t *testing.T) {
		token := am.createSession()
		am.deleteSession(token)
		if am.validateSession(token) {
			t.Error("deleted session should be invalid")
		}
	})

	t.Run("expired session rejected", func(t *testing.T) {
		token := am.createSession()
		am.mu.Lock()
		am.sessions[token] = time.Now().Add(-1 * time.Hour)
		am.mu.Unlock()

		if am.validateSession(token) {
			t.Error("expired session should be invalid")
		}
	})

	t.Run("unique tokens", func(t *testing.T) {
		t1 := am.createSession()
		t2 := am.createSession()
		if t1 == t2 {
			t.Error("tokens should be unique")
		}
	})
}

func TestSessionCookie(t *testing.T) {
	am := newAuthManager()

	t.Run("set cookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		am.setSessionCookie(w, "test-token")
		cookies := w.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("no cookies set")
		}
		c := cookies[0]
		if c.Name != "mansio_session" {
			t.Errorf("cookie name = %q, want %q", c.Name, "mansio_session")
		}
		if c.Value != "test-token" {
			t.Errorf("cookie value = %q, want %q", c.Value, "test-token")
		}
		if !c.HttpOnly {
			t.Error("cookie should be HttpOnly")
		}
	})

	t.Run("clear cookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		am.clearSessionCookie(w)
		cookies := w.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("no cookies set")
		}
		if cookies[0].MaxAge != -1 {
			t.Errorf("MaxAge = %d, want -1", cookies[0].MaxAge)
		}
	})

	t.Run("get token from request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: "mansio_session", Value: "my-token"})
		token := am.getTokenFromRequest(req)
		if token != "my-token" {
			t.Errorf("token = %q, want %q", token, "my-token")
		}
	})

	t.Run("get token from request without cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		token := am.getTokenFromRequest(req)
		if token != "" {
			t.Errorf("token = %q, want empty", token)
		}
	})
}

func TestWebSocketTickets(t *testing.T) {
	am := newAuthManager()

	t.Run("consume valid ticket once", func(t *testing.T) {
		ticket := am.createWebSocketTicket()
		if ticket == "" {
			t.Fatal("ticket should not be empty")
		}
		if !am.consumeWebSocketTicket(ticket) {
			t.Fatal("fresh ticket should be valid")
		}
		if am.consumeWebSocketTicket(ticket) {
			t.Fatal("ticket should be single-use")
		}
	})

	t.Run("expired ticket rejected", func(t *testing.T) {
		ticket := am.createWebSocketTicket()
		am.mu.Lock()
		am.wsTickets[ticket] = time.Now().Add(-1 * time.Hour)
		am.mu.Unlock()

		if am.consumeWebSocketTicket(ticket) {
			t.Fatal("expired ticket should be rejected")
		}
	})

	t.Run("empty ticket rejected", func(t *testing.T) {
		if am.consumeWebSocketTicket("") {
			t.Fatal("empty ticket should be rejected")
		}
	})
}
