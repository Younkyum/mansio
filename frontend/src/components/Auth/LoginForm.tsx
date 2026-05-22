import { useState } from 'react';
import { useEffectiveTheme } from '../../hooks/useEffectiveTheme';

interface LoginFormProps {
  needsSetup: boolean;
  onSuccess: () => void;
}

export function LoginForm({ needsSetup, onSuccess }: LoginFormProps) {
  const { ui } = useEffectiveTheme();
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!password) return;

    setLoading(true);
    setError('');

    const endpoint = needsSetup ? '/api/v1/auth/setup' : '/api/v1/auth/login';
    const res = await fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password }),
    });

    if (res.ok) {
      onSuccess();
    } else {
      const data = await res.json();
      setError(data.error || 'Authentication failed');
    }
    setLoading(false);
  };

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh',
      width: '100vw',
      backgroundColor: ui.appBg,
      fontFamily: "'JetBrains Mono', monospace",
      padding: '24px',
      boxSizing: 'border-box',
    }}>
      <form onSubmit={handleSubmit} style={{
        display: 'flex',
        flexDirection: 'column',
        gap: '16px',
        width: '100%',
        maxWidth: '320px',
      }}>
        <div style={{
          color: ui.textPrimary,
          fontSize: '20px',
          fontWeight: 600,
          textAlign: 'center',
          marginBottom: '8px',
        }}>
          Mansio
        </div>

        <div style={{
          color: ui.textSecondary,
          fontSize: '13px',
          textAlign: 'center',
        }}>
          {needsSetup ? 'Set a password to secure your terminal.' : 'Enter your password to continue.'}
        </div>

        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder={needsSetup ? 'Choose a password' : 'Password'}
          autoFocus
          style={{
            backgroundColor: ui.tabActiveBg,
            border: `1px solid ${ui.sidebarBorder}`,
            color: ui.textPrimary,
            fontSize: '16px',
            fontFamily: 'inherit',
            padding: '12px 14px',
            outline: 'none',
            width: '100%',
            boxSizing: 'border-box',
            borderRadius: 4,
          }}
        />

        {error && (
          <div style={{ color: ui.danger, fontSize: '12px', textAlign: 'center' }}>
            {error}
          </div>
        )}

        <button
          type="submit"
          disabled={loading || !password}
          style={{
            backgroundColor: ui.accent,
            border: 'none',
            color: '#fff',
            fontSize: '14px',
            fontFamily: 'inherit',
            fontWeight: 600,
            padding: '12px',
            cursor: loading ? 'wait' : 'pointer',
            opacity: loading || !password ? 0.6 : 1,
            borderRadius: 4,
            minHeight: 44,
          }}
        >
          {needsSetup ? 'Set Password' : 'Login'}
        </button>
      </form>
    </div>
  );
}
