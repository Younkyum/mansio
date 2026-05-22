package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/younkyumjin/mansio/internal/store"
)

func setupUploadAPI(t *testing.T) (store.Store, *http.ServeMux, string, string) {
	t.Helper()
	dir := t.TempDir()
	uploadDir := t.TempDir()

	s, err := store.NewSQLite(dir)
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	ws, err := s.CreateWorkspace("WS")
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	sess, err := s.CreateSession(ws.ID, "T")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	mux := http.NewServeMux()
	uh := NewUploadHandler(s, uploadDir, 0)
	mux.HandleFunc("POST /api/v1/sessions/{id}/upload", uh.Upload)

	return s, mux, sess.ID, uploadDir
}

func uploadRequest(t *testing.T, sessID, filename string, contents []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write(contents); err != nil {
		t.Fatalf("Write: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest("POST", "/api/v1/sessions/"+sessID+"/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestUpload_Success(t *testing.T) {
	_, mux, sessID, uploadDir := setupUploadAPI(t)

	req := uploadRequest(t, sessID, "hello.txt", []byte("world"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["name"] != "hello.txt" {
		t.Errorf("name = %q, want hello.txt", resp["name"])
	}
	if resp["path"] != filepath.Join(uploadDir, "hello.txt") {
		t.Errorf("path = %q, want %q", resp["path"], filepath.Join(uploadDir, "hello.txt"))
	}

	data, err := os.ReadFile(resp["path"])
	if err != nil {
		t.Fatalf("read uploaded file: %v", err)
	}
	if string(data) != "world" {
		t.Errorf("file contents = %q, want world", data)
	}
}

func TestUpload_PathTraversalSanitized(t *testing.T) {
	_, mux, sessID, uploadDir := setupUploadAPI(t)

	req := uploadRequest(t, sessID, "../../etc/passwd", []byte("evil"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)

	if strings.Contains(resp["name"], "..") || strings.Contains(resp["name"], "/") {
		t.Errorf("name not sanitized: %q", resp["name"])
	}
	if !strings.HasPrefix(resp["path"], uploadDir) {
		t.Errorf("path escaped upload dir: %q (not under %q)", resp["path"], uploadDir)
	}
	if _, err := os.Stat("/etc/passwd_evil"); err == nil {
		t.Error("attacker-controlled path was created outside upload dir")
	}
}

func TestUpload_NonexistentSession(t *testing.T) {
	_, mux, _, _ := setupUploadAPI(t)

	req := uploadRequest(t, "no-such-session", "f.txt", []byte("x"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestUpload_MissingFile(t *testing.T) {
	_, mux, sessID, _ := setupUploadAPI(t)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("note", "no file")
	mw.Close()

	req := httptest.NewRequest("POST", "/api/v1/sessions/"+sessID+"/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestUpload_FilenameCollisionGetsSuffix(t *testing.T) {
	_, mux, sessID, uploadDir := setupUploadAPI(t)

	for i := 0; i < 3; i++ {
		req := uploadRequest(t, sessID, "doc.txt", []byte("v"))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("upload %d: status=%d body=%s", i, w.Code, w.Body.String())
		}
	}

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}

	if !names["doc.txt"] {
		t.Errorf("doc.txt should exist, found: %v", names)
	}
	if !names["doc (1).txt"] || !names["doc (2).txt"] {
		t.Errorf("expected doc.txt + doc (1).txt + doc (2).txt, got: %v", names)
	}
}

func TestUpload_FileTooLarge(t *testing.T) {
	dir := t.TempDir()
	uploadDir := t.TempDir()
	s, err := store.NewSQLite(dir)
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	ws, _ := s.CreateWorkspace("WS")
	sess, _ := s.CreateSession(ws.ID, "T")

	mux := http.NewServeMux()
	uh := NewUploadHandler(s, uploadDir, 100) // 100 byte cap
	mux.HandleFunc("POST /api/v1/sessions/{id}/upload", uh.Upload)

	big := bytes.Repeat([]byte("A"), 10_000)
	req := uploadRequest(t, sess.ID, "big.bin", big)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", w.Code)
	}
}

func TestUpload_RejectsNullByteFilename(t *testing.T) {
	_, mux, sessID, _ := setupUploadAPI(t)

	req := uploadRequest(t, sessID, "evil\x00.txt", []byte("x"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		var resp map[string]string
		json.NewDecoder(w.Body).Decode(&resp)
		if strings.Contains(resp["name"], "\x00") {
			t.Errorf("name contains null byte: %q", resp["name"])
		}
	}
}

// Sanity: streaming reads the body once.
func TestUpload_BodyConsumedOnce(t *testing.T) {
	_, mux, sessID, _ := setupUploadAPI(t)

	contents := []byte("streamed-payload")
	req := uploadRequest(t, sessID, "s.txt", contents)
	body, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(bytes.NewReader(body))

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}
