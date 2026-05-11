import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, render, cleanup } from '@testing-library/react';
import { useAppStore, __resetActivityTrackingForTests } from '../../stores/appStore';
import { TerminalPanel } from './TerminalPanel';

// Count how many times TerminalView is invoked. The point of the selector
// subscription is that TerminalView should NOT re-render when sessionActivity
// (or any other unrelated store slice) mutates — otherwise xterm's in-progress
// text selection gets disrupted.
const viewRenderCount = { value: 0 };

vi.mock('./TerminalView', () => ({
  TerminalView: (props: { sessionId: string | null }) => {
    viewRenderCount.value += 1;
    return <div data-testid="terminal-view" data-session={props.sessionId ?? ''} />;
  },
}));

vi.mock('./TabBar', () => ({
  TabBar: () => <div data-testid="tab-bar" />,
}));

vi.mock('./MobileInputBar', () => ({
  MobileInputBar: () => <div data-testid="mobile-input-bar" />,
}));

describe('TerminalPanel — selector subscription', () => {
  beforeEach(() => {
    viewRenderCount.value = 0;
    __resetActivityTrackingForTests();
    useAppStore.setState({
      workspaces: [],
      sessions: {},
      activeWorkspaceId: null,
      activeSessionId: 'sess-A',
      activeSessionByWorkspace: {},
      sessionActivity: {},
      toasts: [],
      initialized: true,
    });
  });

  afterEach(() => {
    cleanup();
  });

  it('does NOT re-render TerminalView when an unrelated store slice (sessionActivity) mutates', () => {
    render(<TerminalPanel />);
    const baseline = viewRenderCount.value;
    expect(baseline).toBeGreaterThan(0);

    // Mutate a slice TerminalPanel doesn't select on.
    act(() => {
      useAppStore.setState((s) => ({
        sessionActivity: {
          ...s.sessionActivity,
          'sess-A': { unread: true, lastOutputAt: 1, notifiedAt: 1 },
        },
      }));
    });

    expect(viewRenderCount.value).toBe(baseline);
  });

  it('DOES re-render TerminalView when activeSessionId changes', () => {
    render(<TerminalPanel />);
    const baseline = viewRenderCount.value;

    act(() => {
      useAppStore.setState({ activeSessionId: 'sess-B' });
    });

    expect(viewRenderCount.value).toBeGreaterThan(baseline);
  });
});
