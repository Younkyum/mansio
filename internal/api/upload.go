package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/younkyumjin/mansio/internal/store"
)

const (
	defaultMaxUploadBytes int64 = 100 << 20 // 100 MiB
	multipartParseBuffer  int64 = 32 << 20  // 32 MiB in-memory before spilling to tempfile
)

type UploadHandler struct {
	store     store.Store
	uploadDir string
	maxBytes  int64
}

// NewUploadHandler returns an upload endpoint handler.
// uploadDir defaults to ~/uploads when empty.
// maxBytes defaults to 100 MiB when zero.
func NewUploadHandler(s store.Store, uploadDir string, maxBytes int64) *UploadHandler {
	if uploadDir == "" {
		home, _ := os.UserHomeDir()
		uploadDir = filepath.Join(home, "uploads")
	}
	if maxBytes <= 0 {
		maxBytes = defaultMaxUploadBytes
	}
	return &UploadHandler{store: s, uploadDir: uploadDir, maxBytes: maxBytes}
}

func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	sessID := r.PathValue("id")
	sess, err := h.store.GetSession(sessID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sess == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes)

	if err := r.ParseMultipartForm(multipartParseBuffer); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || strings.Contains(err.Error(), "request body too large") {
			writeError(w, http.StatusRequestEntityTooLarge, "file exceeds maximum size")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'file' field")
		return
	}
	defer file.Close()

	safeName := sanitizeUploadFilename(header.Filename)
	if safeName == "" {
		writeError(w, http.StatusBadRequest, "invalid filename")
		return
	}

	if err := os.MkdirAll(h.uploadDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "create upload dir: "+err.Error())
		return
	}

	finalPath, finalName, err := uniqueUploadPath(h.uploadDir, safeName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out, err := os.OpenFile(finalPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create file: "+err.Error())
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		os.Remove(finalPath)
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "file exceeds maximum size")
			return
		}
		writeError(w, http.StatusInternalServerError, "write file: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"path": finalPath,
		"name": finalName,
	})
}

// sanitizeUploadFilename strips path components and dangerous characters.
// Returns "" when no usable name remains.
func sanitizeUploadFilename(raw string) string {
	if strings.ContainsRune(raw, '\x00') {
		return ""
	}
	name := filepath.Base(raw)
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.TrimLeft(name, ".")
	if name == "" || name == "/" {
		return ""
	}
	return name
}

// uniqueUploadPath returns an unused path inside dir, appending " (N)" before
// the extension when the desired name already exists.
func uniqueUploadPath(dir, name string) (string, string, error) {
	candidate := filepath.Join(dir, name)
	if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
		return candidate, name, nil
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; i < 10000; i++ {
		alt := fmt.Sprintf("%s (%d)%s", base, i, ext)
		full := filepath.Join(dir, alt)
		if _, err := os.Stat(full); errors.Is(err, os.ErrNotExist) {
			return full, alt, nil
		}
	}
	return "", "", errors.New("could not find unique upload filename")
}
