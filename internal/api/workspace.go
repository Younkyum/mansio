package api

import (
	"encoding/json"
	"net/http"

	"github.com/younkyumjin/mansio/internal/model"
	"github.com/younkyumjin/mansio/internal/store"
)

type WorkspaceHandler struct {
	store         store.Store
	onDeleteFunc  func(workspaceID string)
}

func NewWorkspaceHandler(s store.Store, onDelete func(workspaceID string)) *WorkspaceHandler {
	return &WorkspaceHandler{store: s, onDeleteFunc: onDelete}
}

func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	workspaces, err := h.store.ListWorkspaces()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if workspaces == nil {
		workspaces = []model.Workspace{}
	}
	writeJSON(w, http.StatusOK, workspaces)
}

func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	workspace, err := h.store.CreateWorkspace(req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, workspace)
}

func (h *WorkspaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	workspace, err := h.store.UpdateWorkspace(id, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if workspace == nil {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}

func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Kill tmux sessions for all tabs before DB cascade delete
	sessions, _ := h.store.ListSessions(id)
	if err := h.store.DeleteWorkspace(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h.onDeleteFunc != nil {
		for _, sess := range sessions {
			h.onDeleteFunc(sess.ID)
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
