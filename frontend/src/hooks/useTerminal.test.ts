import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, cleanup } from '@testing-library/react';
import { useRef } from 'react';

// Per-instance spies — captured by the @xterm mocks below and inspected by tests.
interface FakeTerminal {
  id: number;
  element: HTMLElement | null;
  rows: number;
  options: { theme: unknown };
  open: ReturnType<typeof vi.fn>;
  loadAddon: ReturnType<typeof vi.fn>;
  refresh: ReturnType<typeof vi.fn>;
  onData: ReturnType<typeof vi.fn>;
  onBinary: ReturnType<typeof vi.fn>;
  attachCustomKeyEventHandler: ReturnType<typeof vi.fn>;
  dispose: ReturnType<typeof vi.fn>;
  paste: ReturnType<typeof vi.fn>;
}

const created: FakeTerminal[] = [];

vi.mock('@xterm/xterm', () => {
  let nextId = 0;
  return {
    Terminal: class {
      id = nextId++;
      element: HTMLElement | null = null;
      rows = 24;
      options = { theme: undefined };
      open = vi.fn((container: HTMLElement) => {
        const el = document.createElement('div');
        el.dataset.fake = 'xterm';
        container.appendChild(el);
        this.element = el;
      });
      loadAddon = vi.fn();
      refresh = vi.fn();
      onData = vi.fn();
      onBinary = vi.fn();
      attachCustomKeyEventHandler = vi.fn();
      dispose = vi.fn();
      paste = vi.fn();
      constructor() {
        created.push(this as unknown as FakeTerminal);
      }
    },
  };
});

vi.mock('@xterm/addon-fit', () => ({
  FitAddon: class {
    fit = vi.fn();
    proposeDimensions = vi.fn(() => ({ cols: 80, rows: 24 }));
  },
}));

vi.mock('@xterm/addon-webgl', () => ({
  WebglAddon: class {
    onContextLoss = vi.fn();
    dispose = vi.fn();
    clearTextureAtlas = vi.fn();
  },
}));

vi.mock('@xterm/addon-web-links', () => ({
  WebLinksAddon: class {},
}));

class FakeWebSocket {
  static OPEN = 1;
  static CLOSED = 3;
  readyState = FakeWebSocket.OPEN;
  binaryType = 'arraybuffer';
  onopen: ((this: FakeWebSocket) => void) | null = null;
  onmessage: ((e: { data: unknown }) => void) | null = null;
  onclose: (() => void) | null = null;
  send = vi.fn();
  close = vi.fn();
  constructor(public url: string) {}
}

class FakeResizeObserver {
  observe = vi.fn();
  disconnect = vi.fn();
  unobserve = vi.fn();
  constructor(public cb: ResizeObserverCallback) {}
}

// useTerminal stores instances in a module-level Map; reset between tests.
import { useTerminal, disposeTerminal, nextReconnectDelay } from './useTerminal';

function harness(sessionId: string | null) {
  return renderHook(
    ({ sid }: { sid: string | null }) => {
      const ref = useRef<HTMLDivElement | null>(null);
      // The hook reads ref.current before doing anything; provide a stable container.
      if (!ref.current) {
        const div = document.createElement('div');
        document.body.appendChild(div);
        (ref as { current: HTMLDivElement }).current = div;
      }
      useTerminal({ sessionId: sid, containerRef: ref });
      return ref.current;
    },
    { initialProps: { sid: sessionId } },
  );
}

describe('useTerminal — VS Code detach/attach pattern', () => {
  beforeEach(() => {
    created.length = 0;
    (globalThis as unknown as { WebSocket: typeof FakeWebSocket }).WebSocket = FakeWebSocket;
    (globalThis as unknown as { ResizeObserver: typeof FakeResizeObserver }).ResizeObserver =
      FakeResizeObserver;
  });

  afterEach(() => {
    cleanup();
    // Drain the module-level instance cache so each test starts fresh.
    disposeTerminal('A');
    disposeTerminal('B');
  });

  it('opens a terminal on first mount and does not call refresh', () => {
    const { result } = harness('A');
    expect(created).toHaveLength(1);
    const t = created[0];
    expect(t.open).toHaveBeenCalledTimes(1);
    expect(t.element?.parentElement).toBe(result.current);
    expect(t.refresh).not.toHaveBeenCalled();
  });

  it('A → B detaches A from the container and opens B', () => {
    const { result, rerender } = harness('A');
    const a = created[0];
    const container = result.current!;
    expect(a.element?.parentElement).toBe(container);

    rerender({ sid: 'B' });

    expect(created).toHaveLength(2);
    const b = created[1];
    expect(a.element?.parentElement).toBeNull();
    expect(b.element?.parentElement).toBe(container);
  });

  it('A → B → A re-attaches the cached A element AND triggers refresh', () => {
    const { result, rerender } = harness('A');
    const a = created[0];
    const container = result.current!;

    rerender({ sid: 'B' });
    const b = created[1];

    rerender({ sid: 'A' });

    expect(b.element?.parentElement).toBeNull();
    expect(a.element?.parentElement).toBe(container);
    // The fix: xterm doesn't auto-redraw after DOM re-attach, so refresh must be called.
    expect(a.refresh).toHaveBeenCalledWith(0, a.rows - 1);
    // We did not re-open A — it's still the cached instance.
    expect(a.open).toHaveBeenCalledTimes(1);
  });
});

describe('nextReconnectDelay (exponential backoff with jitter)', () => {
  it('grows as 500 * 2^attempts, capped at 10s', () => {
    // Force jitter to zero (Math.random()=0.5 → jitter coefficient 0).
    const spy = vi.spyOn(Math, 'random').mockReturnValue(0.5);
    try {
      expect(nextReconnectDelay(0)).toBe(500);
      expect(nextReconnectDelay(1)).toBe(1000);
      expect(nextReconnectDelay(2)).toBe(2000);
      expect(nextReconnectDelay(3)).toBe(4000);
      expect(nextReconnectDelay(4)).toBe(8000);
      expect(nextReconnectDelay(5)).toBe(10_000); // cap holds
      expect(nextReconnectDelay(6)).toBe(10_000);
      expect(nextReconnectDelay(99)).toBe(10_000);
    } finally {
      spy.mockRestore();
    }
  });

  it('floors at 250ms even with worst-case negative jitter', () => {
    const spy = vi.spyOn(Math, 'random').mockReturnValue(0); // jitter coeff = -1
    try {
      const d = nextReconnectDelay(0);
      // base=500, jitter=-100, raw=400; floor of 250 must keep us above 250.
      expect(d).toBeGreaterThanOrEqual(250);
      expect(d).toBeLessThanOrEqual(500);
    } finally {
      spy.mockRestore();
    }
  });

  it('produces a spread when many tabs reconnect at once (jitter desyncs them)', () => {
    // 100 reconnect timings at attempt=2 should not all be identical.
    const delays = new Set<number>();
    for (let i = 0; i < 100; i++) delays.add(nextReconnectDelay(2));
    expect(delays.size).toBeGreaterThan(1);
  });
});
