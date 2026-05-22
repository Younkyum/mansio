import { useEffect, useState, useRef, useCallback } from 'react';
import { useAppStore } from './stores/appStore';
import { Sidebar } from './components/Sidebar/Sidebar';
import { TerminalPanel } from './components/Terminal/TerminalPanel';
import { LoginForm } from './components/Auth/LoginForm';
import { ToastContainer } from './components/Toast/ToastContainer';
import { useEffectiveTheme } from './hooks/useEffectiveTheme';
import { useMediaQuery, MOBILE_QUERY } from './hooks/useMediaQuery';

type AuthState = 'loading' | 'needs-setup' | 'needs-login' | 'authenticated';

export default function App() {
  const initialized = useAppStore((s) => s.initialized);
  const init = useAppStore((s) => s.init);
  const pollActive = useAppStore((s) => s.pollActive);
  const { ui } = useEffectiveTheme();
  const isMobile = useMediaQuery(MOBILE_QUERY);

  const PERM_DISMISS_KEY = 'mansio.permissionBannerDismissed';

  const [authState, setAuthState] = useState<AuthState>('loading');
  const [permWarning, setPermWarning] = useState<string | null>(null);
  const [permChecking, setPermChecking] = useState(false);
  const [permDismissed, setPermDismissed] = useState<boolean>(() => {
    try {
      return localStorage.getItem(PERM_DISMISS_KEY) === '1';
    } catch {
      return false;
    }
  });
  const [sidebarWidth, setSidebarWidth] = useState(220);
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const dragging = useRef(false);
  const rafId = useRef<number | undefined>(undefined);
  const latestX = useRef<number>(0);

  const checkAuth = useCallback(async () => {
    const res = await fetch('/api/v1/auth/check');
    const data = await res.json();

    if (data.needsSetup) {
      setAuthState('needs-setup');
    } else {
      const testRes = await fetch('/api/v1/workspaces');
      if (testRes.status === 401) {
        setAuthState('needs-login');
      } else {
        setAuthState('authenticated');
      }
    }
  }, []);

  const checkPermissions = useCallback(async () => {
    setPermChecking(true);
    try {
      const res = await fetch('/api/v1/health');
      const data = await res.json();
      if (data.permissions === false && data.permissionMessage) {
        setPermWarning(data.permissionMessage);
      } else {
        setPermWarning(null);
        // Permissions are healthy — clear the dismiss flag so the banner
        // can reappear if permissions later break again.
        try {
          localStorage.removeItem(PERM_DISMISS_KEY);
        } catch {}
        setPermDismissed(false);
      }
    } catch {}
    setPermChecking(false);
  }, []);

  const dismissPermBanner = useCallback(() => {
    try {
      localStorage.setItem(PERM_DISMISS_KEY, '1');
    } catch {}
    setPermDismissed(true);
  }, []);

  const onMenuClick = useCallback(() => setMobileSidebarOpen(true), []);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  useEffect(() => {
    if (authState === 'authenticated' && !initialized) {
      init();
      checkPermissions();
    }
  }, [authState, initialized, init, checkPermissions]);

  useEffect(() => {
    if (!isMobile) setMobileSidebarOpen(false);
  }, [isMobile]);

  // Poll the active workspace's session list every 5s so the sidebar's CWD
  // subtitle reflects `cd` activity. Pauses when the tab is hidden — no point
  // refreshing data the user can't see.
  useEffect(() => {
    if (!initialized) return;
    const tick = () => {
      if (document.visibilityState === 'visible') {
        pollActive();
      }
    };
    const id = window.setInterval(tick, 5000);
    return () => window.clearInterval(id);
  }, [initialized, pollActive]);

  // getState() — the listener only needs activeSessionId at the moment
  // visibility flips, so don't subscribe.
  useEffect(() => {
    const onVisibility = () => {
      if (document.visibilityState !== 'visible') return;
      const { activeSessionId, clearSessionUnread } = useAppStore.getState();
      if (activeSessionId) clearSessionUnread(activeSessionId);
    };
    document.addEventListener('visibilitychange', onVisibility);
    return () => document.removeEventListener('visibilitychange', onVisibility);
  }, []);

  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (!dragging.current) return;

    latestX.current = e.clientX;
    if (rafId.current === undefined) {
      rafId.current = requestAnimationFrame(() => {
        const newWidth = Math.max(140, Math.min(400, latestX.current));
        setSidebarWidth(newWidth);
        rafId.current = undefined;
      });
    }
  }, []);

  const onMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    dragging.current = true;

    const onMouseUp = () => {
      dragging.current = false;
      if (rafId.current !== undefined) {
        cancelAnimationFrame(rafId.current);
        rafId.current = undefined;
      }
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', onMouseUp);
    };
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', onMouseUp);
  }, [handleMouseMove]);

  if (authState === 'loading') {
    return (
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        backgroundColor: ui.appBg,
        color: ui.textPrimary,
        fontFamily: "'JetBrains Mono', monospace",
        padding: '24px',
      }}>
        Starting Mansio...
      </div>
    );
  }

  if (authState === 'needs-setup' || authState === 'needs-login') {
    return (
      <LoginForm
        needsSetup={authState === 'needs-setup'}
        onSuccess={() => setAuthState('authenticated')}
      />
    );
  }

  if (!initialized) {
    return (
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        backgroundColor: ui.appBg,
        color: ui.textPrimary,
        fontFamily: "'JetBrains Mono', monospace",
        padding: '24px',
      }}>
        Loading...
      </div>
    );
  }

  const sidebarPanelWidth = isMobile ? Math.min(280, window.innerWidth - 56) : sidebarWidth;

  const showPermBanner = !!permWarning && !permDismissed;

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      width: '100vw',
      height: '100dvh',
      backgroundColor: ui.appBg,
      overflow: 'hidden',
      position: 'relative',
    }}>
      {showPermBanner && (
        <div style={{
          flexShrink: 0,
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          padding: '8px 12px',
          backgroundColor: ui.warning,
          color: '#000',
          fontFamily: "'JetBrains Mono', monospace",
          fontSize: 12,
          lineHeight: 1.4,
          borderBottom: `1px solid ${ui.sidebarBorder}`,
        }}>
          <span style={{ fontWeight: 600, flexShrink: 0 }}>Permission</span>
          <span style={{ flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            Full Disk Access required — System Settings ▸ Privacy & Security ▸ Full Disk Access ▸ add <code style={{ background: 'rgba(0,0,0,0.12)', padding: '0 4px', borderRadius: 2 }}>/usr/local/bin/mansio</code>
          </span>
          <button
            onClick={checkPermissions}
            disabled={permChecking}
            style={{
              flexShrink: 0,
              padding: '4px 10px',
              backgroundColor: 'rgba(0,0,0,0.15)',
              border: '1px solid rgba(0,0,0,0.25)',
              color: '#000',
              fontSize: 12,
              fontFamily: 'inherit',
              fontWeight: 600,
              cursor: permChecking ? 'wait' : 'pointer',
              opacity: permChecking ? 0.6 : 1,
              borderRadius: 3,
            }}
          >
            {permChecking ? 'Checking…' : 'Recheck'}
          </button>
          <button
            onClick={dismissPermBanner}
            aria-label="Dismiss"
            title="Dismiss"
            style={{
              flexShrink: 0,
              width: 22,
              height: 22,
              padding: 0,
              backgroundColor: 'transparent',
              border: 'none',
              color: '#000',
              fontSize: 16,
              lineHeight: 1,
              fontFamily: 'inherit',
              cursor: 'pointer',
              borderRadius: 3,
            }}
          >
            ×
          </button>
        </div>
      )}

      <div style={{
        display: 'flex',
        flex: 1,
        minHeight: 0,
        position: 'relative',
      }}>
      {!isMobile && (
        <>
          <div style={{ width: sidebarPanelWidth, flexShrink: 0, height: '100%' }}>
            <Sidebar />
          </div>

          <div
            onMouseDown={onMouseDown}
            style={{
              width: '4px',
              cursor: 'col-resize',
              backgroundColor: 'transparent',
              flexShrink: 0,
            }}
            onMouseEnter={(e) => {
              (e.target as HTMLElement).style.backgroundColor = ui.accent;
            }}
            onMouseLeave={(e) => {
              if (!dragging.current) {
                (e.target as HTMLElement).style.backgroundColor = 'transparent';
              }
            }}
          />
        </>
      )}

      {isMobile && mobileSidebarOpen && (
        <>
          <div
            onClick={() => setMobileSidebarOpen(false)}
            style={{
              position: 'absolute',
              inset: 0,
              backgroundColor: ui.overlayBg,
              zIndex: 999,
            }}
            aria-hidden
          />
          <div style={{
            position: 'absolute',
            top: 0,
            left: 0,
            bottom: 0,
            width: sidebarPanelWidth,
            zIndex: 1000,
            boxShadow: '2px 0 8px rgba(0,0,0,0.35)',
          }}>
            <Sidebar onCloseMobile={() => setMobileSidebarOpen(false)} />
          </div>
        </>
      )}

      <div style={{ flex: 1, height: '100%', overflow: 'hidden', minWidth: 0 }}>
        <TerminalPanel
          showMenuButton={isMobile}
          onMenuClick={onMenuClick}
        />
      </div>
      </div>
      <ToastContainer />
    </div>
  );
}
