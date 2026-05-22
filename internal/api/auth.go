package api

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/younkyumjin/mansio/internal/store"
)

type AuthHandler struct {
	store        store.Store
	createSession func() string
	setCookie    func(http.ResponseWriter, string)
	clearCookie  func(http.ResponseWriter)
	getToken     func(*http.Request) string
	deleteSession func(string)
}

type AuthHandlerConfig struct {
	Store         store.Store
	CreateSession func() string
	SetCookie     func(http.ResponseWriter, string)
	ClearCookie   func(http.ResponseWriter)
	GetToken      func(*http.Request) string
	DeleteSession func(string)
}

func NewAuthHandler(cfg AuthHandlerConfig) *AuthHandler {
	return &AuthHandler{
		store:         cfg.Store,
		createSession: cfg.CreateSession,
		setCookie:     cfg.SetCookie,
		clearCookie:   cfg.ClearCookie,
		getToken:      cfg.GetToken,
		deleteSession: cfg.DeleteSession,
	}
}

func (h *AuthHandler) Check(w http.ResponseWriter, r *http.Request) {
	hasPass, _ := h.store.HasPassword()
	writeJSON(w, http.StatusOK, map[string]bool{
		"authenticated": true,
		"needsSetup":    !hasPass,
	})
}

func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	hasPass, _ := h.store.HasPassword()
	if hasPass {
		writeError(w, http.StatusBadRequest, "password already configured")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := h.store.SetPasswordHash(string(hash)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save password")
		return
	}

	token := h.createSession()
	h.setCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	hash, err := h.store.GetPasswordHash()
	if err != nil || hash == "" {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token := h.createSession()
	h.setCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := h.getToken(r)
	if token != "" {
		h.deleteSession(token)
	}
	h.clearCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
