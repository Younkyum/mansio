import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useAppStore, sessionUnread, workspaceUnread } from './appStore';

const { mockWorkspace, mockSession } = vi.hoisted(() => ({
  mockWorkspace: { id: 'ws-1', name: 'Test', sortOrder: 0, createdAt: '', updatedAt: '' },
  mockSession: { id: 's-1', workspaceId: 'ws-1', title: 'Terminal', sortOrder: 0, createdAt: '', updatedAt: '' },
}));

vi.mock('../api/client', () => ({
  api: {
    listWorkspaces: vi.fn().mockResolvedValue([]),
    createWorkspace: vi.fn().mockResolvedValue(mockWorkspace),
    updateWorkspace: vi.fn().mockResolvedValue({ ...mockWorkspace, name: 'Renamed' }),
    deleteWorkspace: vi.fn().mockResolvedValue({ ok: true }),
    listSessions: vi.fn().mockResolvedValue([]),
    createSession: vi.fn().mockResolvedValue(mockSession),
    updateSession: vi.fn().mockResolvedValue({ ...mockSession, title: 'Updated' }),
    deleteSession: vi.fn().mockResolvedValue({ ok: true }),
  },
}));

function setVisibility(state: 'visible' | 'hidden') {
  Object.defineProperty(document, 'visibilityState', {
    configurable: true,
    get: () => state,
  });
}

describe('appStore', () => {
  beforeEach(() => {
    useAppStore.setState({
      workspaces: [],
      sessions: {},
      activeWorkspaceId: null,
      activeSessionId: null,
      activeSessionByWorkspace: {},
      sessionActivity: {},
      initialized: false,
    });
    setVisibility('visible');
  });

  it('starts uninitialized', () => {
    const state = useAppStore.getState();
    expect(state.initialized).toBe(false);
    expect(state.workspaces).toEqual([]);
    expect(state.activeWorkspaceId).toBeNull();
    expect(state.activeSessionId).toBeNull();
  });

  it('createWorkspace adds workspace and creates session', async () => {
    await useAppStore.getState().createWorkspace('Test');
    const state = useAppStore.getState();
    expect(state.workspaces).toHaveLength(1);
    expect(state.workspaces[0].name).toBe('Test');
    expect(state.activeWorkspaceId).toBe('ws-1');
    expect(state.activeSessionId).toBe('s-1');
    expect(state.sessions['ws-1']).toHaveLength(1);
  });

  it('deleteWorkspace removes workspace and sessions', async () => {
    useAppStore.setState({
      workspaces: [mockWorkspace],
      sessions: { 'ws-1': [mockSession] },
      activeWorkspaceId: 'ws-1',
      activeSessionId: 's-1',
    });

    await useAppStore.getState().deleteWorkspace('ws-1');
    const state = useAppStore.getState();
    expect(state.workspaces).toHaveLength(0);
    expect(state.sessions['ws-1']).toBeUndefined();
    expect(state.activeWorkspaceId).toBeNull();
  });

  it('renameWorkspace updates the workspace name', async () => {
    useAppStore.setState({
      workspaces: [mockWorkspace],
      activeWorkspaceId: 'ws-1',
    });

    await useAppStore.getState().renameWorkspace('ws-1', 'Renamed');
    const ws = useAppStore.getState().workspaces[0];
    expect(ws.name).toBe('Renamed');
  });

  it('setActiveSession updates activeSessionId', () => {
    useAppStore.getState().setActiveSession('s-2');
    expect(useAppStore.getState().activeSessionId).toBe('s-2');
  });

  it('deleteSession updates active to next', async () => {
    const s2 = { ...mockSession, id: 's-2', title: 'Second' };
    useAppStore.setState({
      workspaces: [mockWorkspace],
      sessions: { 'ws-1': [mockSession, s2] },
      activeWorkspaceId: 'ws-1',
      activeSessionId: 's-1',
    });

    await useAppStore.getState().deleteSession('s-1');
    const state = useAppStore.getState();
    expect(state.sessions['ws-1']).toHaveLength(1);
    expect(state.activeSessionId).toBe('s-2');
  });

  describe('sessionActivity', () => {
    it('markSessionOutput sets unread when sessionId is not the active session', () => {
      useAppStore.setState({ activeSessionId: 's-1' });
      useAppStore.getState().markSessionOutput('s-2');
      const state = useAppStore.getState();
      expect(sessionUnread(state, 's-2')).toBe(true);
      expect(state.sessionActivity['s-2'].lastOutputAt).toBeGreaterThan(0);
    });

    it('markSessionOutput is a no-op when sessionId is active and document is visible', () => {
      useAppStore.setState({ activeSessionId: 's-1' });
      setVisibility('visible');
      useAppStore.getState().markSessionOutput('s-1');
      const state = useAppStore.getState();
      expect(sessionUnread(state, 's-1')).toBe(false);
      expect(state.sessionActivity['s-1']).toBeUndefined();
    });

    it('markSessionOutput sets unread when sessionId is active but document is hidden', () => {
      useAppStore.setState({ activeSessionId: 's-1' });
      setVisibility('hidden');
      useAppStore.getState().markSessionOutput('s-1');
      expect(sessionUnread(useAppStore.getState(), 's-1')).toBe(true);
    });

    it('setActiveSession clears unread for the newly-active session', () => {
      useAppStore.setState({
        activeWorkspaceId: 'ws-1',
        activeSessionId: 's-1',
        sessionActivity: {
          's-2': { unread: true, lastOutputAt: 100, notifiedAt: 0 },
        },
      });
      useAppStore.getState().setActiveSession('s-2');
      const state = useAppStore.getState();
      expect(state.activeSessionId).toBe('s-2');
      expect(sessionUnread(state, 's-2')).toBe(false);
      // Timestamps preserved — only `unread` flips.
      expect(state.sessionActivity['s-2'].lastOutputAt).toBe(100);
    });

    it('clearSessionUnread leaves lastOutputAt and notifiedAt untouched', () => {
      useAppStore.setState({
        sessionActivity: {
          's-1': { unread: true, lastOutputAt: 100, notifiedAt: 50 },
        },
      });
      useAppStore.getState().clearSessionUnread('s-1');
      const a = useAppStore.getState().sessionActivity['s-1'];
      expect(a.unread).toBe(false);
      expect(a.lastOutputAt).toBe(100);
      expect(a.notifiedAt).toBe(50);
    });

    it('workspaceUnread is true when any session in that workspace has unread', () => {
      const s2 = { ...mockSession, id: 's-2' };
      useAppStore.setState({
        workspaces: [mockWorkspace],
        sessions: { 'ws-1': [mockSession, s2] },
        sessionActivity: {
          's-2': { unread: true, lastOutputAt: 1, notifiedAt: 0 },
        },
      });
      expect(workspaceUnread(useAppStore.getState(), 'ws-1')).toBe(true);
    });

    it('workspaceUnread is false when no session in that workspace has unread', () => {
      useAppStore.setState({
        workspaces: [mockWorkspace],
        sessions: { 'ws-1': [mockSession] },
        sessionActivity: {
          's-1': { unread: false, lastOutputAt: 1, notifiedAt: 0 },
        },
      });
      expect(workspaceUnread(useAppStore.getState(), 'ws-1')).toBe(false);
    });

    it('deleteSession prunes its sessionActivity entry', async () => {
      const s2 = { ...mockSession, id: 's-2' };
      useAppStore.setState({
        workspaces: [mockWorkspace],
        sessions: { 'ws-1': [mockSession, s2] },
        activeWorkspaceId: 'ws-1',
        activeSessionId: 's-1',
        sessionActivity: {
          's-2': { unread: true, lastOutputAt: 1, notifiedAt: 0 },
        },
      });
      await useAppStore.getState().deleteSession('s-2');
      expect(useAppStore.getState().sessionActivity['s-2']).toBeUndefined();
    });
  });
});
