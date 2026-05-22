package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/younkyumjin/mansio/internal/model"
)

const schema = `
CREATE TABLE IF NOT EXISTS workspaces (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title        TEXT NOT NULL DEFAULT 'Terminal',
    shell        TEXT NOT NULL DEFAULT '',
    sort_order   INTEGER NOT NULL DEFAULT 0,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_workspace ON sessions(workspace_id);

CREATE TABLE IF NOT EXISTS auth_config (
    id            INTEGER PRIMARY KEY CHECK (id = 1),
    password_hash TEXT NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLite(dataDir string) (*SQLiteStore, error) {
	dbPath := filepath.Join(dataDir, "mansio.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Workspaces

func (s *SQLiteStore) ListWorkspaces() ([]model.Workspace, error) {
	rows, err := s.db.Query("SELECT id, name, sort_order, created_at, updated_at FROM workspaces ORDER BY sort_order, created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []model.Workspace
	for rows.Next() {
		var w model.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.SortOrder, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, rows.Err()
}

func (s *SQLiteStore) GetWorkspace(id string) (*model.Workspace, error) {
	var w model.Workspace
	err := s.db.QueryRow("SELECT id, name, sort_order, created_at, updated_at FROM workspaces WHERE id = ?", id).
		Scan(&w.ID, &w.Name, &w.SortOrder, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *SQLiteStore) CreateWorkspace(name string) (*model.Workspace, error) {
	w := model.Workspace{
		ID:        uuid.New().String(),
		Name:      name,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	var maxOrder int
	s.db.QueryRow("SELECT COALESCE(MAX(sort_order), -1) FROM workspaces").Scan(&maxOrder)
	w.SortOrder = maxOrder + 1

	_, err := s.db.Exec(
		"INSERT INTO workspaces (id, name, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		w.ID, w.Name, w.SortOrder, w.CreatedAt, w.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *SQLiteStore) UpdateWorkspace(id, name string) (*model.Workspace, error) {
	now := time.Now().UTC()
	result, err := s.db.Exec("UPDATE workspaces SET name = ?, updated_at = ? WHERE id = ?", name, now, id)
	if err != nil {
		return nil, err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, nil
	}
	return s.GetWorkspace(id)
}

func (s *SQLiteStore) DeleteWorkspace(id string) error {
	_, err := s.db.Exec("DELETE FROM workspaces WHERE id = ?", id)
	return err
}

// Sessions

func (s *SQLiteStore) ListSessions(workspaceID string) ([]model.Session, error) {
	rows, err := s.db.Query(
		"SELECT id, workspace_id, title, shell, sort_order, created_at, updated_at FROM sessions WHERE workspace_id = ? ORDER BY sort_order, created_at",
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []model.Session
	for rows.Next() {
		var sess model.Session
		if err := rows.Scan(&sess.ID, &sess.WorkspaceID, &sess.Title, &sess.Shell, &sess.SortOrder, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

func (s *SQLiteStore) GetSession(id string) (*model.Session, error) {
	var sess model.Session
	err := s.db.QueryRow(
		"SELECT id, workspace_id, title, shell, sort_order, created_at, updated_at FROM sessions WHERE id = ?", id,
	).Scan(&sess.ID, &sess.WorkspaceID, &sess.Title, &sess.Shell, &sess.SortOrder, &sess.CreatedAt, &sess.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *SQLiteStore) CreateSession(workspaceID, title string) (*model.Session, error) {
	if title == "" {
		title = "Terminal"
	}

	sess := model.Session{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		Title:       title,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	var maxOrder int
	s.db.QueryRow("SELECT COALESCE(MAX(sort_order), -1) FROM sessions WHERE workspace_id = ?", workspaceID).Scan(&maxOrder)
	sess.SortOrder = maxOrder + 1

	_, err := s.db.Exec(
		"INSERT INTO sessions (id, workspace_id, title, shell, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		sess.ID, sess.WorkspaceID, sess.Title, sess.Shell, sess.SortOrder, sess.CreatedAt, sess.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *SQLiteStore) UpdateSession(id, title string) (*model.Session, error) {
	now := time.Now().UTC()
	result, err := s.db.Exec("UPDATE sessions SET title = ?, updated_at = ? WHERE id = ?", title, now, id)
	if err != nil {
		return nil, err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, nil
	}
	return s.GetSession(id)
}

func (s *SQLiteStore) DeleteSession(id string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

// Auth

func (s *SQLiteStore) HasPassword() (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM auth_config").Scan(&count)
	return count > 0, err
}

func (s *SQLiteStore) GetPasswordHash() (string, error) {
	var hash string
	err := s.db.QueryRow("SELECT password_hash FROM auth_config WHERE id = 1").Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return hash, err
}

func (s *SQLiteStore) SetPasswordHash(hash string) error {
	_, err := s.db.Exec(
		"INSERT INTO auth_config (id, password_hash) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET password_hash = ?",
		hash, hash,
	)
	return err
}
