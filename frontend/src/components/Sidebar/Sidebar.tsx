import { useState, useRef, useEffect } from 'react';
import { useAppStore } from '../../stores/appStore';
import { useEffectiveTheme } from '../../hooks/useEffectiveTheme';
import type { ThemeMode } from '../../lib/theme';

interface ContextMenu {
  workspaceId: string;
  x: number;
  y: number;
}

interface SidebarProps {
  onCloseMobile?: () => void;
}

const themeIcon: Record<ThemeMode, string> = {
  system: '🖥',
  light: '☀',
  dark: '☾',
};

const themeLabel: Record<ThemeMode, string> = {
  system: 'System',
  light: 'Light',
  dark: 'Dark',
};

export function Sidebar({ onCloseMobile }: SidebarProps) {
  const {
    workspaces,
    sessions,
    activeWorkspaceId,
    activeSessionByWorkspace,
    setActiveWorkspace,
    createWorkspace,
    deleteWorkspace,
    renameWorkspace,
  } = useAppStore();
  const sessionActivity = useAppStore((s) => s.sessionActivity);
  const { ui, mode, cycleMode } = useEffectiveTheme();

  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState('');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState('');
  const [contextMenu, setContextMenu] = useState<ContextMenu | null>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setContextMenu(null);
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  const handleCreate = async () => {
    if (newName.trim()) {
      await createWorkspace(newName.trim());
      setNewName('');
      setCreating(false);
    }
  };

  const handleRename = async (id: string) => {
    if (editName.trim()) {
      await renameWorkspace(id, editName.trim());
    }
    setEditingId(null);
  };

  const startRename = (id: string, name: string) => {
    setContextMenu(null);
    setEditingId(id);
    setEditName(name);
  };

  const handleContextMenu = (e: React.MouseEvent, wsId: string) => {
    e.preventDefault();
    e.stopPropagation();
    setContextMenu({ workspaceId: wsId, x: e.clientX, y: e.clientY });
  };

  return (
    <div style={{
      width: '100%',
      height: '100%',
      backgroundColor: ui.sidebarBg,
      borderRight: `1px solid ${ui.sidebarBorder}`,
      display: 'flex',
      flexDirection: 'column',
      fontFamily: "'JetBrains Mono', monospace",
    }}>
      <div style={{
        padding: '12px 16px',
        fontSize: '11px',
        fontWeight: 600,
        letterSpacing: '0.5px',
        textTransform: 'uppercase',
        color: ui.textMuted,
        borderBottom: `1px solid ${ui.sidebarBorder}`,
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        gap: 8,
      }}>
        <span>Workspaces</span>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <button
            onClick={() => setCreating(true)}
            style={{
              background: 'none',
              border: 'none',
              color: ui.textSecondary,
              cursor: 'pointer',
              fontSize: '18px',
              padding: '4px 6px',
              lineHeight: 1,
              minHeight: 28,
              minWidth: 28,
            }}
            title="New Workspace"
            aria-label="New Workspace"
          >+</button>
          {onCloseMobile && (
            <button
              onClick={onCloseMobile}
              style={{
                background: 'none',
                border: 'none',
                color: ui.textSecondary,
                cursor: 'pointer',
                fontSize: '18px',
                padding: '4px 8px',
                lineHeight: 1,
                minHeight: 28,
                minWidth: 28,
              }}
              title="Close"
              aria-label="Close sidebar"
            >×</button>
          )}
        </div>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '4px 0' }}>
        {workspaces.map((ws) => {
          // Pick the CWD of the workspace's last-active session; fall back to
          // the first session if we haven't tracked one yet.
          const wsSessions = sessions[ws.id] ?? [];
          const activeId = activeSessionByWorkspace[ws.id];
          const activeSession =
            wsSessions.find((s) => s.id === activeId) ?? wsSessions[0];
          const cwd = activeSession?.cwd;
          const isActive = ws.id === activeWorkspaceId;
          const hasUnread =
            !isActive && wsSessions.some((s) => sessionActivity[s.id]?.unread === true);
          return (
            <div
              key={ws.id}
              onClick={() => setActiveWorkspace(ws.id)}
              onContextMenu={(e) => handleContextMenu(e, ws.id)}
              style={{
                padding: '10px 16px',
                cursor: 'pointer',
                backgroundColor: ws.id === activeWorkspaceId ? ui.tabActiveBg : 'transparent',
                color: ws.id === activeWorkspaceId ? ui.textPrimary : ui.textSecondary,
                fontSize: '13px',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                minHeight: 40,
              }}
            >
              {editingId === ws.id ? (
                <input
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  onBlur={() => handleRename(ws.id)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleRename(ws.id);
                    if (e.key === 'Escape') setEditingId(null);
                  }}
                  autoFocus
                  onClick={(e) => e.stopPropagation()}
                  style={{
                    background: 'transparent',
                    border: `1px solid ${ui.accent}`,
                    color: ui.textPrimary,
                    fontSize: '13px',
                    fontFamily: 'inherit',
                    padding: '4px 6px',
                    width: '100%',
                    outline: 'none',
                  }}
                />
              ) : (
                <div
                  onDoubleClick={(e) => {
                    e.stopPropagation();
                    startRename(ws.id, ws.name);
                  }}
                  style={{
                    flex: 1,
                    minWidth: 0,
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 2,
                  }}
                  title="Double-click or right-click to rename"
                >
                  <span
                    style={{
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {ws.name}
                  </span>
                  {cwd && (
                    <span
                      style={{
                        fontSize: '11px',
                        color: ui.textMuted,
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                        // Clip the start of long paths so the tail (the
                        // current dir) stays visible.
                        direction: 'rtl',
                        textAlign: 'left',
                      }}
                      title={cwd}
                    >
                      {cwd}
                    </span>
                  )}
                </div>
              )}
              {hasUnread && (
                <span
                  aria-label="unread output"
                  title="New output in this workspace"
                  style={{
                    width: 6,
                    height: 6,
                    borderRadius: '50%',
                    backgroundColor: ui.accent,
                    flexShrink: 0,
                    marginLeft: 8,
                  }}
                />
              )}
            </div>
          );
        })}

        {creating && (
          <div style={{ padding: '4px 16px' }}>
            <input
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleCreate();
                if (e.key === 'Escape') { setCreating(false); setNewName(''); }
              }}
              onBlur={() => { if (!newName.trim()) setCreating(false); }}
              autoFocus
              placeholder="Workspace name..."
              style={{
                width: '100%',
                background: 'transparent',
                border: `1px solid ${ui.accent}`,
                color: ui.textPrimary,
                fontSize: '13px',
                fontFamily: 'inherit',
                padding: '8px 10px',
                outline: 'none',
                boxSizing: 'border-box',
              }}
            />
          </div>
        )}
      </div>

      <div style={{
        padding: '10px 12px',
        borderTop: `1px solid ${ui.sidebarBorder}`,
        fontSize: '11px',
        color: ui.textMuted,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 8,
      }}>
        <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          LociTerm
        </span>
        <button
          onClick={cycleMode}
          style={{
            background: 'transparent',
            border: `1px solid ${ui.sidebarBorder}`,
            borderRadius: 4,
            color: ui.textSecondary,
            cursor: 'pointer',
            fontFamily: 'inherit',
            fontSize: 11,
            padding: '4px 8px',
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            minHeight: 28,
          }}
          title={`Theme: ${themeLabel[mode]} (click to change)`}
          aria-label={`Theme: ${themeLabel[mode]}, click to change`}
        >
          <span aria-hidden>{themeIcon[mode]}</span>
          <span>{themeLabel[mode]}</span>
        </button>
      </div>

      {contextMenu && (
        <div
          ref={menuRef}
          style={{
            position: 'fixed',
            left: contextMenu.x,
            top: contextMenu.y,
            backgroundColor: ui.tabActiveBg,
            border: `1px solid ${ui.sidebarBorder}`,
            zIndex: 1000,
            minWidth: '140px',
            fontFamily: "'JetBrains Mono', monospace",
            fontSize: '12px',
          }}
        >
          <button
            onClick={() => {
              const ws = workspaces.find(w => w.id === contextMenu.workspaceId);
              if (ws) startRename(ws.id, ws.name);
            }}
            style={{
              display: 'block',
              width: '100%',
              padding: '8px 12px',
              background: 'none',
              border: 'none',
              color: ui.textPrimary,
              cursor: 'pointer',
              textAlign: 'left',
              fontFamily: 'inherit',
              fontSize: 'inherit',
            }}
            onMouseEnter={(e) => { (e.target as HTMLElement).style.backgroundColor = ui.hoverBg; }}
            onMouseLeave={(e) => { (e.target as HTMLElement).style.backgroundColor = 'transparent'; }}
          >
            Rename
          </button>
          {workspaces.length > 1 && (
            <button
              onClick={() => {
                deleteWorkspace(contextMenu.workspaceId);
                setContextMenu(null);
              }}
              style={{
                display: 'block',
                width: '100%',
                padding: '8px 12px',
                background: 'none',
                border: 'none',
                color: ui.danger,
                cursor: 'pointer',
                textAlign: 'left',
                fontFamily: 'inherit',
                fontSize: 'inherit',
              }}
              onMouseEnter={(e) => { (e.target as HTMLElement).style.backgroundColor = ui.hoverBg; }}
              onMouseLeave={(e) => { (e.target as HTMLElement).style.backgroundColor = 'transparent'; }}
            >
              Delete
            </button>
          )}
        </div>
      )}
    </div>
  );
}
