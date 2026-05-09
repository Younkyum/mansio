import { create } from 'zustand';
import type { Workspace, Session } from '../types';
import { api } from '../api/client';

export interface SessionActivity {
  unread: boolean;
  lastOutputAt: number;
  notifiedAt: number;
}

interface AppState {
  workspaces: Workspace[];
  sessions: Record<string, Session[]>;
  activeWorkspaceId: string | null;
  activeSessionId: string | null;
  // Last active session per workspace. Used both to restore the right tab when
  // switching workspaces and to pick which session's CWD the sidebar shows.
  activeSessionByWorkspace: Record<string, string>;
  sessionActivity: Record<string, SessionActivity>;
  initialized: boolean;

  init: () => Promise<void>;
  fetchWorkspaces: () => Promise<void>;
  createWorkspace: (name: string) => Promise<void>;
  deleteWorkspace: (id: string) => Promise<void>;
  renameWorkspace: (id: string, name: string) => Promise<void>;
  setActiveWorkspace: (id: string) => Promise<void>;

  fetchSessions: (workspaceId: string) => Promise<void>;
  pollActive: () => Promise<void>;
  createSession: (workspaceId: string, title?: string) => Promise<void>;
  deleteSession: (id: string) => Promise<void>;
  renameSession: (id: string, title: string) => Promise<void>;
  setActiveSession: (id: string) => void;

  markSessionOutput: (sessionId: string) => void;
  clearSessionUnread: (sessionId: string) => void;
}

export const useAppStore = create<AppState>((set, get) => ({
  workspaces: [],
  sessions: {},
  activeWorkspaceId: null,
  activeSessionId: null,
  activeSessionByWorkspace: {},
  sessionActivity: {},
  initialized: false,

  init: async () => {
    let workspaces = await api.listWorkspaces();

    if (workspaces.length === 0) {
      const ws = await api.createWorkspace('Default');
      workspaces = [ws];
    }

    // Fetch all workspaces' sessions in parallel so the sidebar can show CWD
    // for every workspace, not just the active one.
    const lists = await Promise.all(workspaces.map((w) => api.listSessions(w.id)));
    const sessions: Record<string, Session[]> = {};
    const activeSessionByWorkspace: Record<string, string> = {};
    workspaces.forEach((w, i) => {
      sessions[w.id] = lists[i];
      if (lists[i].length > 0) {
        activeSessionByWorkspace[w.id] = lists[i][0].id;
      }
    });

    const wid = workspaces[0].id;
    if (sessions[wid].length === 0) {
      const sess = await api.createSession(wid);
      sessions[wid] = [sess];
      activeSessionByWorkspace[wid] = sess.id;
    }

    set({
      workspaces,
      sessions,
      activeWorkspaceId: wid,
      activeSessionId: activeSessionByWorkspace[wid] ?? null,
      activeSessionByWorkspace,
      initialized: true,
    });
  },

  fetchWorkspaces: async () => {
    const workspaces = await api.listWorkspaces();
    set({ workspaces });
  },

  createWorkspace: async (name: string) => {
    const ws = await api.createWorkspace(name);
    const sess = await api.createSession(ws.id);
    set((s) => ({
      workspaces: [...s.workspaces, ws],
      sessions: { ...s.sessions, [ws.id]: [sess] },
      activeWorkspaceId: ws.id,
      activeSessionId: sess.id,
      activeSessionByWorkspace: { ...s.activeSessionByWorkspace, [ws.id]: sess.id },
    }));
  },

  deleteWorkspace: async (id: string) => {
    await api.deleteWorkspace(id);
    set((s) => {
      const workspaces = s.workspaces.filter((w) => w.id !== id);
      const removedSessions = s.sessions[id] ?? [];
      const sessions = { ...s.sessions };
      delete sessions[id];
      const activeSessionByWorkspace = { ...s.activeSessionByWorkspace };
      delete activeSessionByWorkspace[id];
      const nextWid = workspaces.length > 0 ? workspaces[0].id : null;
      const nextActive = nextWid ? activeSessionByWorkspace[nextWid] ?? sessions[nextWid]?.[0]?.id ?? null : null;
      const sessionActivity = { ...s.sessionActivity };
      for (const sess of removedSessions) {
        delete sessionActivity[sess.id];
      }
      return {
        workspaces,
        sessions,
        activeWorkspaceId: nextWid,
        activeSessionId: nextActive,
        activeSessionByWorkspace,
        sessionActivity,
      };
    });
  },

  renameWorkspace: async (id: string, name: string) => {
    const updated = await api.updateWorkspace(id, name);
    set((s) => ({
      workspaces: s.workspaces.map((w) => (w.id === id ? updated : w)),
    }));
  },

  setActiveWorkspace: async (id: string) => {
    const state = get();
    let sessions = state.sessions[id];
    if (!sessions) {
      sessions = await api.listSessions(id);
    }
    set((s) => {
      const remembered = s.activeSessionByWorkspace[id];
      const stillExists = remembered && sessions.some((sess) => sess.id === remembered);
      const nextActive = stillExists ? remembered : sessions[0]?.id ?? null;
      return {
        sessions: { ...s.sessions, [id]: sessions },
        activeWorkspaceId: id,
        activeSessionId: nextActive,
        activeSessionByWorkspace: nextActive
          ? { ...s.activeSessionByWorkspace, [id]: nextActive }
          : s.activeSessionByWorkspace,
      };
    });
  },

  fetchSessions: async (workspaceId: string) => {
    const sessions = await api.listSessions(workspaceId);
    set((s) => ({
      sessions: { ...s.sessions, [workspaceId]: sessions },
    }));
  },

  // Refresh the active workspace's session list — primarily to pick up CWD
  // changes for the sidebar subtitle. Called on a 5s interval from App.tsx.
  pollActive: async () => {
    const wid = get().activeWorkspaceId;
    if (!wid) return;
    const sessions = await api.listSessions(wid);
    set((s) => ({
      sessions: { ...s.sessions, [wid]: sessions },
    }));
  },

  createSession: async (workspaceId: string, title?: string) => {
    const sess = await api.createSession(workspaceId, title);
    set((s) => ({
      sessions: {
        ...s.sessions,
        [workspaceId]: [...(s.sessions[workspaceId] || []), sess],
      },
      activeSessionId: sess.id,
      activeSessionByWorkspace: { ...s.activeSessionByWorkspace, [workspaceId]: sess.id },
    }));
  },

  deleteSession: async (id: string) => {
    await api.deleteSession(id);
    set((s) => {
      const wid = s.activeWorkspaceId;
      if (!wid) return s;
      const sessions = (s.sessions[wid] || []).filter((sess) => sess.id !== id);
      const wasActive = s.activeSessionId === id;
      const nextActive = wasActive ? sessions[0]?.id ?? null : s.activeSessionId;
      const activeSessionByWorkspace = { ...s.activeSessionByWorkspace };
      if (activeSessionByWorkspace[wid] === id) {
        if (nextActive) {
          activeSessionByWorkspace[wid] = nextActive;
        } else {
          delete activeSessionByWorkspace[wid];
        }
      }
      const sessionActivity = { ...s.sessionActivity };
      delete sessionActivity[id];
      return {
        sessions: { ...s.sessions, [wid]: sessions },
        activeSessionId: nextActive,
        activeSessionByWorkspace,
        sessionActivity,
      };
    });
  },

  renameSession: async (id: string, title: string) => {
    const updated = await api.updateSession(id, title);
    set((s) => {
      const wid = s.activeWorkspaceId;
      if (!wid) return s;
      return {
        sessions: {
          ...s.sessions,
          [wid]: (s.sessions[wid] || []).map((sess) =>
            sess.id === id ? updated : sess
          ),
        },
      };
    });
  },

  setActiveSession: (id: string) => {
    set((s) => {
      const wid = s.activeWorkspaceId;
      const prev = s.sessionActivity[id];
      const sessionActivity =
        prev && prev.unread
          ? { ...s.sessionActivity, [id]: { ...prev, unread: false } }
          : s.sessionActivity;
      return {
        activeSessionId: id,
        activeSessionByWorkspace: wid
          ? { ...s.activeSessionByWorkspace, [wid]: id }
          : s.activeSessionByWorkspace,
        sessionActivity,
      };
    });
  },

  // Skip when the session is the active foreground — the user is already
  // watching it, so there's nothing to badge.
  markSessionOutput: (sessionId: string) => {
    const state = get();
    const isActiveForeground =
      sessionId === state.activeSessionId &&
      typeof document !== 'undefined' &&
      document.visibilityState === 'visible';
    if (isActiveForeground) return;
    const now = Date.now();
    set((s) => {
      const prev = s.sessionActivity[sessionId];
      return {
        sessionActivity: {
          ...s.sessionActivity,
          [sessionId]: {
            unread: true,
            lastOutputAt: now,
            notifiedAt: prev?.notifiedAt ?? 0,
          },
        },
      };
    });
  },

  clearSessionUnread: (sessionId: string) => {
    set((s) => {
      const prev = s.sessionActivity[sessionId];
      if (!prev || !prev.unread) return s;
      return {
        sessionActivity: {
          ...s.sessionActivity,
          [sessionId]: { ...prev, unread: false },
        },
      };
    });
  },
}));

export function sessionUnread(state: AppState, sessionId: string): boolean {
  return state.sessionActivity[sessionId]?.unread === true;
}

export function workspaceUnread(state: AppState, workspaceId: string): boolean {
  const list = state.sessions[workspaceId];
  if (!list) return false;
  for (const s of list) {
    if (state.sessionActivity[s.id]?.unread === true) return true;
  }
  return false;
}
