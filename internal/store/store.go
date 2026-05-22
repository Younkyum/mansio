package store

import "github.com/younkyumjin/mansio/internal/model"

type Store interface {
	// Workspaces
	ListWorkspaces() ([]model.Workspace, error)
	GetWorkspace(id string) (*model.Workspace, error)
	CreateWorkspace(name string) (*model.Workspace, error)
	UpdateWorkspace(id, name string) (*model.Workspace, error)
	DeleteWorkspace(id string) error

	// Auth
	GetPasswordHash() (string, error)
	SetPasswordHash(hash string) error
	HasPassword() (bool, error)

	// Sessions
	ListSessions(workspaceID string) ([]model.Session, error)
	GetSession(id string) (*model.Session, error)
	CreateSession(workspaceID, title string) (*model.Session, error)
	UpdateSession(id, title string) (*model.Session, error)
	DeleteSession(id string) error

	Close() error
}
