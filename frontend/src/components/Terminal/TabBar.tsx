import { useAppStore } from '../../stores/appStore';
import { useEffectiveTheme } from '../../hooks/useEffectiveTheme';

interface TabBarProps {
  showMenuButton?: boolean;
  onMenuClick?: () => void;
}

export function TabBar({ showMenuButton, onMenuClick }: TabBarProps) {
  const {
    activeWorkspaceId,
    activeSessionId,
    sessions,
    setActiveSession,
    createSession,
    deleteSession,
  } = useAppStore();
  const sessionActivity = useAppStore((s) => s.sessionActivity);
  const { ui } = useEffectiveTheme();

  if (!activeWorkspaceId) return null;

  const currentSessions = sessions[activeWorkspaceId] || [];

  return (
    <div style={{
      display: 'flex',
      alignItems: 'stretch',
      backgroundColor: ui.tabBarBg,
      borderBottom: `1px solid ${ui.tabBorder}`,
      height: '40px',
      fontFamily: "'JetBrains Mono', monospace",
      fontSize: '12px',
      overflow: 'hidden',
    }}>
      {showMenuButton && (
        <button
          onClick={onMenuClick}
          style={{
            background: 'none',
            border: 'none',
            borderRight: `1px solid ${ui.tabBorder}`,
            color: ui.textSecondary,
            cursor: 'pointer',
            fontSize: '18px',
            padding: '0 14px',
            display: 'flex',
            alignItems: 'center',
            minWidth: 44,
          }}
          aria-label="Open sidebar"
          title="Workspaces"
        >☰</button>
      )}

      <div style={{
        display: 'flex',
        flex: 1,
        overflowX: 'auto',
        overflowY: 'hidden',
      }}>
        {currentSessions.map((sess) => {
          const isActive = sess.id === activeSessionId;
          const hasUnread = !isActive && sessionActivity[sess.id]?.unread === true;
          return (
          <div
            key={sess.id}
            onClick={() => setActiveSession(sess.id)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              padding: '0 12px',
              cursor: 'pointer',
              backgroundColor: isActive ? ui.tabActiveBg : 'transparent',
              color: isActive ? ui.textPrimary : ui.textSecondary,
              borderRight: `1px solid ${ui.tabBorder}`,
              whiteSpace: 'nowrap',
              minWidth: 0,
              maxWidth: 200,
              flexShrink: 0,
            }}
          >
            {hasUnread && (
              <span
                aria-label="unread output"
                title="New output"
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  backgroundColor: ui.accent,
                  flexShrink: 0,
                }}
              />
            )}
            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {sess.title}
            </span>
            {currentSessions.length > 1 && (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  deleteSession(sess.id);
                }}
                style={{
                  background: 'none',
                  border: 'none',
                  color: ui.textMuted,
                  cursor: 'pointer',
                  fontSize: '16px',
                  padding: '4px 6px',
                  lineHeight: 1,
                  opacity: sess.id === activeSessionId ? 0.7 : 0.3,
                }}
                aria-label={`Close ${sess.title}`}
              >×</button>
            )}
          </div>
          );
        })}
      </div>

      <button
        onClick={() => activeWorkspaceId && createSession(activeWorkspaceId)}
        style={{
          background: 'none',
          border: 'none',
          borderLeft: `1px solid ${ui.tabBorder}`,
          color: ui.textSecondary,
          cursor: 'pointer',
          fontSize: '18px',
          padding: '0 14px',
          display: 'flex',
          alignItems: 'center',
          minWidth: 44,
        }}
        title="New Tab"
        aria-label="New tab"
      >+</button>
    </div>
  );
}
