package api

import (
	"encoding/json"
	"net/http"

	"github.com/younkyumjin/mansio/internal/model"
	"github.com/younkyumjin/mansio/internal/store"
)

type SessionHandler struct {
	store        store.Store
	onDeleteFunc func(sessionID string)
	getCwdFunc   func(sessionID string) string
}

func NewSessionHandler(s store.Store, getCwd func(sessionID string) string, onDelete func(sessionID string)) *SessionHandler {
	return &SessionHandler{store: s, getCwdFunc: getCwd, onDeleteFunc: onDelete}
}

func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("wid")
	sessions, err := h.store.ListSessions(wid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sessions == nil {
		sessions = []model.Session{}
	}
	if h.getCwdFunc != nil {
		for i := range sessions {
			sessions[i].Cwd = h.getCwdFunc(sessions[i].ID)
		}
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("wid")
	var req struct {
		Title string `json:"title"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	session, err := h.store.CreateSession(wid, req.Title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

func (h *SessionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session, err := h.store.UpdateSession(id, req.Title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (h *SessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteSession(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h.onDeleteFunc != nil {
		h.onDeleteFunc(id)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
