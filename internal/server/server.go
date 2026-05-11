package server

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/younkyumjin/lociterm/internal/api"
	"github.com/younkyumjin/lociterm/internal/store"
	"github.com/younkyumjin/lociterm/internal/tmux"
	"github.com/younkyumjin/lociterm/internal/ws"
)

type Server struct {
	frontendFS fs.FS
	store      store.Store
	tmuxMgr    *tmux.Manager
	auth       *authManager
}

func New(frontendFS fs.FS, dataDir string) *Server {
	s, err := store.NewSQLite(dataDir)
	if err != nil {
		log.Fatalf("failed to open store: %v", err)
	}

	return &Server{
		frontendFS: frontendFS,
		store:      s,
		tmuxMgr:    tmux.NewManager(dataDir),
		auth:       newAuthManager(),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", s.handleHealth)

	ah := api.NewAuthHandler(api.AuthHandlerConfig{
		Store:         s.store,
		CreateSession: s.auth.createSession,
		SetCookie:     s.auth.setSessionCookie,
		ClearCookie:   s.auth.clearSessionCookie,
		GetToken:      s.auth.getTokenFromRequest,
		DeleteSession: s.auth.deleteSession,
	})
	mux.HandleFunc("GET /api/v1/auth/check", ah.Check)
	mux.HandleFunc("POST /api/v1/auth/setup", ah.Setup)
	mux.HandleFunc("POST /api/v1/auth/login", ah.Login)
	mux.HandleFunc("POST /api/v1/auth/logout", ah.Logout)

	killTmux := func(sessionID string) {
		s.tmuxMgr.KillSession(sessionID)
	}

	wh := api.NewWorkspaceHandler(s.store, killTmux)
	mux.HandleFunc("GET /api/v1/workspaces", s.requireAuth(wh.List))
	mux.HandleFunc("POST /api/v1/workspaces", s.requireAuth(wh.Create))
	mux.HandleFunc("PATCH /api/v1/workspaces/{id}", s.requireAuth(wh.Update))
	mux.HandleFunc("DELETE /api/v1/workspaces/{id}", s.requireAuth(wh.Delete))

	sh := api.NewSessionHandler(s.store, s.tmuxMgr.GetCwd, killTmux)
	mux.HandleFunc("GET /api/v1/workspaces/{wid}/sessions", s.requireAuth(sh.List))
	mux.HandleFunc("POST /api/v1/workspaces/{wid}/sessions", s.requireAuth(sh.Create))
	mux.HandleFunc("PATCH /api/v1/sessions/{id}", s.requireAuth(sh.Update))
	mux.HandleFunc("DELETE /api/v1/sessions/{id}", s.requireAuth(sh.Delete))

	uh := api.NewUploadHandler(s.store, "", 0)
	mux.HandleFunc("POST /api/v1/sessions/{id}/upload", s.requireAuth(uh.Upload))

	wsh := ws.NewHandler(s.tmuxMgr)
	mux.HandleFunc("/api/v1/ws/terminal/{sessionId}", s.requireAuth(wsh.HandleTerminal))

	mux.Handle("/", s.spaHandler())

	return mux
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hasPass, _ := s.store.HasPassword()
		if !hasPass {
			next(w, r)
			return
		}

		token := s.auth.getTokenFromRequest(r)
		if token == "" || !s.auth.validateSession(token) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	homeDir, _ := os.UserHomeDir()
	permOk := true
	permMsg := ""

	if runtime.GOOS == "darwin" {
		testDirs := []string{
			filepath.Join(homeDir, "Documents"),
			filepath.Join(homeDir, "Desktop"),
		}
		for _, dir := range testDirs {
			if _, err := os.ReadDir(dir); err != nil {
				permOk = false
				permMsg = "Full Disk Access required. Go to System Settings > Privacy & Security > Full Disk Access and add lociterm (/usr/local/bin/lociterm)."
				break
			}
		}
	}

	resp := map[string]any{
		"status":      "ok",
		"permissions": permOk,
	}
	if permMsg != "" {
		resp["permissionMessage"] = permMsg
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) spaHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.frontendFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if f, err := s.frontendFS.Open(cleanPath); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) Shutdown() {
	if s.tmuxMgr != nil {
		s.tmuxMgr.Shutdown()
	}
	if s.store != nil {
		s.store.Close()
	}
}
