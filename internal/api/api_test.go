package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/younkyumjin/mansio/internal/store"
)

func setupTestAPI(t *testing.T) (store.Store, *http.ServeMux) {
	t.Helper()
	dir := t.TempDir()
	s, err := store.NewSQLite(dir)
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	mux := http.NewServeMux()
	noop := func(string) {}
	wh := NewWorkspaceHandler(s, noop)
	mux.HandleFunc("GET /api/v1/workspaces", wh.List)
	mux.HandleFunc("POST /api/v1/workspaces", wh.Create)
	mux.HandleFunc("PATCH /api/v1/workspaces/{id}", wh.Update)
	mux.HandleFunc("DELETE /api/v1/workspaces/{id}", wh.Delete)

	sh := NewSessionHandler(s, nil, noop)
	mux.HandleFunc("GET /api/v1/workspaces/{wid}/sessions", sh.List)
	mux.HandleFunc("POST /api/v1/workspaces/{wid}/sessions", sh.Create)
	mux.HandleFunc("PATCH /api/v1/sessions/{id}", sh.Update)
	mux.HandleFunc("DELETE /api/v1/sessions/{id}", sh.Delete)

	return s, mux
}

func doRequest(mux *http.ServeMux, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func decodeJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(w.Body).Decode(&v); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, w.Body.String())
	}
	return v
}

func TestWorkspaceAPI(t *testing.T) {
	_, mux := setupTestAPI(t)

	t.Run("list empty", func(t *testing.T) {
		w := doRequest(mux, "GET", "/api/v1/workspaces", nil)
		if w.Code != 200 {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var list []map[string]any
		json.NewDecoder(w.Body).Decode(&list)
		if len(list) != 0 {
			t.Errorf("expected empty list, got %d items", len(list))
		}
	})

	t.Run("create workspace", func(t *testing.T) {
		w := doRequest(mux, "POST", "/api/v1/workspaces", map[string]string{"name": "Test WS"})
		if w.Code != 201 {
			t.Fatalf("status = %d, want 201", w.Code)
		}
		ws := decodeJSON[map[string]any](t, w)
		if ws["name"] != "Test WS" {
			t.Errorf("name = %v, want %q", ws["name"], "Test WS")
		}
		if ws["id"] == nil || ws["id"] == "" {
			t.Error("id should not be empty")
		}
	})

	t.Run("create without name returns 400", func(t *testing.T) {
		w := doRequest(mux, "POST", "/api/v1/workspaces", map[string]string{"name": ""})
		if w.Code != 400 {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("update workspace", func(t *testing.T) {
		cr := doRequest(mux, "POST", "/api/v1/workspaces", map[string]string{"name": "Original"})
		ws := decodeJSON[map[string]any](t, cr)
		id := ws["id"].(string)

		up := doRequest(mux, "PATCH", "/api/v1/workspaces/"+id, map[string]string{"name": "Renamed"})
		if up.Code != 200 {
			t.Fatalf("status = %d, want 200", up.Code)
		}
		updated := decodeJSON[map[string]any](t, up)
		if updated["name"] != "Renamed" {
			t.Errorf("name = %v, want %q", updated["name"], "Renamed")
		}
	})

	t.Run("update nonexistent returns 404", func(t *testing.T) {
		w := doRequest(mux, "PATCH", "/api/v1/workspaces/nonexistent", map[string]string{"name": "X"})
		if w.Code != 404 {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})

	t.Run("delete workspace", func(t *testing.T) {
		cr := doRequest(mux, "POST", "/api/v1/workspaces", map[string]string{"name": "ToDelete"})
		ws := decodeJSON[map[string]any](t, cr)
		id := ws["id"].(string)

		del := doRequest(mux, "DELETE", "/api/v1/workspaces/"+id, nil)
		if del.Code != 200 {
			t.Fatalf("status = %d, want 200", del.Code)
		}
	})
}

func TestSessionAPI(t *testing.T) {
	_, mux := setupTestAPI(t)

	cr := doRequest(mux, "POST", "/api/v1/workspaces", map[string]string{"name": "WS"})
	ws := decodeJSON[map[string]any](t, cr)
	wid := ws["id"].(string)

	t.Run("list empty sessions", func(t *testing.T) {
		w := doRequest(mux, "GET", "/api/v1/workspaces/"+wid+"/sessions", nil)
		if w.Code != 200 {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var list []map[string]any
		json.NewDecoder(w.Body).Decode(&list)
		if len(list) != 0 {
			t.Errorf("expected empty, got %d", len(list))
		}
	})

	t.Run("create session", func(t *testing.T) {
		w := doRequest(mux, "POST", "/api/v1/workspaces/"+wid+"/sessions", map[string]string{"title": "Dev"})
		if w.Code != 201 {
			t.Fatalf("status = %d, want 201", w.Code)
		}
		sess := decodeJSON[map[string]any](t, w)
		if sess["title"] != "Dev" {
			t.Errorf("title = %v, want %q", sess["title"], "Dev")
		}
		if sess["workspaceId"] != wid {
			t.Errorf("workspaceId = %v, want %q", sess["workspaceId"], wid)
		}
	})

	t.Run("create with default title", func(t *testing.T) {
		w := doRequest(mux, "POST", "/api/v1/workspaces/"+wid+"/sessions", nil)
		if w.Code != 201 {
			t.Fatalf("status = %d, want 201", w.Code)
		}
		sess := decodeJSON[map[string]any](t, w)
		if sess["title"] != "Terminal" {
			t.Errorf("title = %v, want %q", sess["title"], "Terminal")
		}
	})

	t.Run("update session", func(t *testing.T) {
		cr := doRequest(mux, "POST", "/api/v1/workspaces/"+wid+"/sessions", map[string]string{"title": "Old"})
		sess := decodeJSON[map[string]any](t, cr)
		id := sess["id"].(string)

		up := doRequest(mux, "PATCH", "/api/v1/sessions/"+id, map[string]string{"title": "New"})
		if up.Code != 200 {
			t.Fatalf("status = %d, want 200", up.Code)
		}
		updated := decodeJSON[map[string]any](t, up)
		if updated["title"] != "New" {
			t.Errorf("title = %v, want %q", updated["title"], "New")
		}
	})

	t.Run("delete session", func(t *testing.T) {
		cr := doRequest(mux, "POST", "/api/v1/workspaces/"+wid+"/sessions", map[string]string{"title": "Kill"})
		sess := decodeJSON[map[string]any](t, cr)
		id := sess["id"].(string)

		del := doRequest(mux, "DELETE", "/api/v1/sessions/"+id, nil)
		if del.Code != 200 {
			t.Fatalf("status = %d, want 200", del.Code)
		}
	})
}

func TestInvalidJSON(t *testing.T) {
	_, mux := setupTestAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/workspaces", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
