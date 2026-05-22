package store

import (
	"os"
	"testing"
)

func setupTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dir := t.TempDir()
	s, err := NewSQLite(dir)
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestWorkspaceCRUD(t *testing.T) {
	s := setupTestStore(t)

	t.Run("create and list", func(t *testing.T) {
		ws, err := s.CreateWorkspace("Project A")
		if err != nil {
			t.Fatalf("CreateWorkspace: %v", err)
		}
		if ws.Name != "Project A" {
			t.Errorf("name = %q, want %q", ws.Name, "Project A")
		}
		if ws.ID == "" {
			t.Error("ID should not be empty")
		}

		list, err := s.ListWorkspaces()
		if err != nil {
			t.Fatalf("ListWorkspaces: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("len = %d, want 1", len(list))
		}
		if list[0].ID != ws.ID {
			t.Errorf("listed ID = %q, want %q", list[0].ID, ws.ID)
		}
	})

	t.Run("get by id", func(t *testing.T) {
		ws, _ := s.CreateWorkspace("Get Test")
		got, err := s.GetWorkspace(ws.ID)
		if err != nil {
			t.Fatalf("GetWorkspace: %v", err)
		}
		if got == nil || got.ID != ws.ID {
			t.Errorf("got %v, want ID %q", got, ws.ID)
		}
	})

	t.Run("get nonexistent returns nil", func(t *testing.T) {
		got, err := s.GetWorkspace("nonexistent-id")
		if err != nil {
			t.Fatalf("GetWorkspace: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("update", func(t *testing.T) {
		ws, _ := s.CreateWorkspace("Old Name")
		updated, err := s.UpdateWorkspace(ws.ID, "New Name")
		if err != nil {
			t.Fatalf("UpdateWorkspace: %v", err)
		}
		if updated.Name != "New Name" {
			t.Errorf("name = %q, want %q", updated.Name, "New Name")
		}
	})

	t.Run("update nonexistent returns nil", func(t *testing.T) {
		got, err := s.UpdateWorkspace("nonexistent", "Name")
		if err != nil {
			t.Fatalf("UpdateWorkspace: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("delete", func(t *testing.T) {
		ws, _ := s.CreateWorkspace("To Delete")
		if err := s.DeleteWorkspace(ws.ID); err != nil {
			t.Fatalf("DeleteWorkspace: %v", err)
		}
		got, _ := s.GetWorkspace(ws.ID)
		if got != nil {
			t.Error("workspace should be deleted")
		}
	})

	t.Run("sort order increments", func(t *testing.T) {
		s2 := setupTestStore(t)
		w1, _ := s2.CreateWorkspace("First")
		w2, _ := s2.CreateWorkspace("Second")
		if w1.SortOrder >= w2.SortOrder {
			t.Errorf("sort orders: %d >= %d", w1.SortOrder, w2.SortOrder)
		}
	})
}

func TestSessionCRUD(t *testing.T) {
	s := setupTestStore(t)
	ws, _ := s.CreateWorkspace("Host")

	t.Run("create and list", func(t *testing.T) {
		sess, err := s.CreateSession(ws.ID, "Terminal 1")
		if err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		if sess.Title != "Terminal 1" {
			t.Errorf("title = %q, want %q", sess.Title, "Terminal 1")
		}
		if sess.WorkspaceID != ws.ID {
			t.Errorf("workspaceID = %q, want %q", sess.WorkspaceID, ws.ID)
		}

		list, err := s.ListSessions(ws.ID)
		if err != nil {
			t.Fatalf("ListSessions: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("len = %d, want 1", len(list))
		}
	})

	t.Run("default title", func(t *testing.T) {
		sess, _ := s.CreateSession(ws.ID, "")
		if sess.Title != "Terminal" {
			t.Errorf("title = %q, want %q", sess.Title, "Terminal")
		}
	})

	t.Run("get by id", func(t *testing.T) {
		sess, _ := s.CreateSession(ws.ID, "Get Test")
		got, err := s.GetSession(sess.ID)
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		if got == nil || got.ID != sess.ID {
			t.Errorf("got %v, want ID %q", got, sess.ID)
		}
	})

	t.Run("update", func(t *testing.T) {
		sess, _ := s.CreateSession(ws.ID, "Old")
		updated, err := s.UpdateSession(sess.ID, "New")
		if err != nil {
			t.Fatalf("UpdateSession: %v", err)
		}
		if updated.Title != "New" {
			t.Errorf("title = %q, want %q", updated.Title, "New")
		}
	})

	t.Run("delete", func(t *testing.T) {
		sess, _ := s.CreateSession(ws.ID, "ToDelete")
		if err := s.DeleteSession(sess.ID); err != nil {
			t.Fatalf("DeleteSession: %v", err)
		}
		got, _ := s.GetSession(sess.ID)
		if got != nil {
			t.Error("session should be deleted")
		}
	})

	t.Run("cascade delete on workspace removal", func(t *testing.T) {
		ws2, _ := s.CreateWorkspace("Cascade")
		sess, _ := s.CreateSession(ws2.ID, "Child")
		s.DeleteWorkspace(ws2.ID)
		got, _ := s.GetSession(sess.ID)
		if got != nil {
			t.Error("session should be cascade deleted")
		}
	})

	t.Run("list only returns sessions for workspace", func(t *testing.T) {
		ws1, _ := s.CreateWorkspace("WS1")
		ws2, _ := s.CreateWorkspace("WS2")
		s.CreateSession(ws1.ID, "S1")
		s.CreateSession(ws2.ID, "S2")

		list1, _ := s.ListSessions(ws1.ID)
		list2, _ := s.ListSessions(ws2.ID)
		if len(list1) != 1 || list1[0].Title != "S1" {
			t.Errorf("ws1 sessions: %v", list1)
		}
		if len(list2) != 1 || list2[0].Title != "S2" {
			t.Errorf("ws2 sessions: %v", list2)
		}
	})
}

func TestAuthConfig(t *testing.T) {
	s := setupTestStore(t)

	t.Run("no password initially", func(t *testing.T) {
		has, err := s.HasPassword()
		if err != nil {
			t.Fatalf("HasPassword: %v", err)
		}
		if has {
			t.Error("should not have password initially")
		}
	})

	t.Run("get empty hash", func(t *testing.T) {
		hash, err := s.GetPasswordHash()
		if err != nil {
			t.Fatalf("GetPasswordHash: %v", err)
		}
		if hash != "" {
			t.Errorf("hash = %q, want empty", hash)
		}
	})

	t.Run("set and get password", func(t *testing.T) {
		if err := s.SetPasswordHash("$2a$10$fakehash"); err != nil {
			t.Fatalf("SetPasswordHash: %v", err)
		}
		has, _ := s.HasPassword()
		if !has {
			t.Error("should have password after set")
		}
		hash, _ := s.GetPasswordHash()
		if hash != "$2a$10$fakehash" {
			t.Errorf("hash = %q, want %q", hash, "$2a$10$fakehash")
		}
	})

	t.Run("update password", func(t *testing.T) {
		s.SetPasswordHash("$2a$10$newhash")
		hash, _ := s.GetPasswordHash()
		if hash != "$2a$10$newhash" {
			t.Errorf("hash = %q, want %q", hash, "$2a$10$newhash")
		}
	})
}

func TestDatabasePersistence(t *testing.T) {
	dir := t.TempDir()

	s1, err := NewSQLite(dir)
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	ws, _ := s1.CreateWorkspace("Persistent")
	s1.CreateSession(ws.ID, "Session1")
	s1.Close()

	s2, err := NewSQLite(dir)
	if err != nil {
		t.Fatalf("NewSQLite reopen: %v", err)
	}
	defer s2.Close()

	list, _ := s2.ListWorkspaces()
	if len(list) != 1 || list[0].Name != "Persistent" {
		t.Errorf("workspaces after reopen: %v", list)
	}

	sessions, _ := s2.ListSessions(ws.ID)
	if len(sessions) != 1 || sessions[0].Title != "Session1" {
		t.Errorf("sessions after reopen: %v", sessions)
	}
}

func TestDatabaseFileCreated(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewSQLite(dir)
	defer s.Close()

	_, err := os.Stat(dir + "/mansio.db")
	if os.IsNotExist(err) {
		t.Error("database file should exist")
	}
}
