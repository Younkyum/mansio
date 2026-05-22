import { useEffect, useRef } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebglAddon } from '@xterm/addon-webgl';
import { WebLinksAddon } from '@xterm/addon-web-links';
import type { ITheme } from '@xterm/xterm';
import { darkTerminalTheme } from '../lib/theme';
import { createShiftEnterHandler } from './shiftEnter';
import { useAppStore } from '../stores/appStore';

interface UseTerminalOptions {
  sessionId: string | null;
  containerRef: React.RefObject<HTMLDivElement | null>;
  theme?: ITheme;
}

interface TerminalInstance {
  terminal: Terminal;
  fitAddon: FitAddon;
  ws: WebSocket | null;
  connecting: boolean;
  webgl: WebglAddon | null;
  reconnectTimer: number | undefined;
  lastSentCols: number | undefined;
  lastSentRows: number | undefined;
  resizeDebounceTimer: number | undefined;
  attached: boolean;
  listenersBound: boolean;
  // IME composition state — see bindInputListeners for why these exist.
  composing: boolean;
  compositionSuppressUntil: number;
  // Reconnect attempt counter for exponential backoff. Reset to 0 on
  // successful ws.onopen. Used by nextReconnectDelay to grow the wait
  // from 500ms up to a 10s cap so a flaky network or backend restart
  // doesn't slam the server (and shrinks the Detach overwrite race
  // window vs. the old fixed 2s reconnect).
  reconnectAttempts: number;
}

// Exponential backoff schedule for WS reconnects: 500ms, 1s, 2s, 4s, 8s, 10s…
// Capped at 10s with ±20% jitter so multiple tabs reconnecting in unison
// don't all fire at the same instant.
export function nextReconnectDelay(attempts: number): number {
  const base = Math.min(10_000, 500 * Math.pow(2, attempts));
  const jitter = base * 0.2 * (Math.random() * 2 - 1);
  return Math.max(250, Math.floor(base + jitter));
}

const instances = new Map<string, TerminalInstance>();

function createInstance(theme: ITheme): TerminalInstance {
  const terminal = new Terminal({
    theme,
    fontFamily: "'JetBrains Mono', 'SF Mono', 'Menlo', 'Noto Sans KR', 'Noto Sans CJK SC', monospace",
    fontSize: 14,
    lineHeight: 1.2,
    cursorBlink: true,
    cursorStyle: 'block',
    allowProposedApi: true,
    scrollback: 10000,
    // Disable scroll easing — the 125ms animation adds perceptible latency on
    // every Enter keystroke (cursor scrolls to bottom with 125ms easing).
    // Trackpad scroll feels slightly choppier without easing, but keystroke
    // responsiveness is the higher priority. (T2.2)
    smoothScrollDuration: 0,
  });
  const fitAddon = new FitAddon();
  terminal.loadAddon(fitAddon);
  terminal.loadAddon(new WebLinksAddon());

  return {
    terminal,
    fitAddon,
    ws: null,
    connecting: false,
    webgl: null,
    reconnectTimer: undefined,
    lastSentCols: undefined,
    lastSentRows: undefined,
    resizeDebounceTimer: undefined,
    attached: false,
    listenersBound: false,
    composing: false,
    compositionSuppressUntil: 0,
    reconnectAttempts: 0,
  };
}

// Per xterm.js WebglAddon README: register onContextLoss to dispose the
// addon when the browser evicts the WebGL context. xterm falls back to its
// built-in renderer automatically. Without this, the canvas goes black and
// stays black for the lifetime of the page.
function loadWebgl(inst: TerminalInstance): void {
  if (inst.webgl) return;
  try {
    const webgl = new WebglAddon();
    webgl.onContextLoss(() => {
      console.warn('WebGL context lost; attempting to reload addon');
      webgl.dispose();
      inst.webgl = null;
      try {
        loadWebgl(inst);
      } catch {
        console.warn('WebGL addon reload failed; using fallback renderer');
      }
    });
    inst.terminal.loadAddon(webgl);
    inst.webgl = webgl;
  } catch {
    console.warn('WebGL addon failed, using fallback renderer');
  }
}

function configureTextarea(inst: TerminalInstance): void {
  const ta = (inst.terminal as unknown as { textarea?: HTMLTextAreaElement }).textarea;
  if (!ta) return;
  // Stop mobile keyboards from autocapitalize/autocorrect/spellcheck on
  // terminal input — particularly important for CJK IME mid-composition.
  ta.setAttribute('autocapitalize', 'off');
  ta.setAttribute('autocorrect', 'off');
  ta.setAttribute('autocomplete', 'off');
  ta.setAttribute('spellcheck', 'false');
}

// Bind input → WebSocket listeners exactly once per instance. The listener
// closures read `inst.ws` at call time so they survive reconnects without
// needing to be re-attached (which would otherwise leak duplicate listeners).
function bindInputListeners(inst: TerminalInstance): void {
  if (inst.listenersBound) return;
  inst.listenersBound = true;

  // WebSocket.send rejects views over SharedArrayBuffer in current TS lib types,
  // so the parameter is narrowed to Uint8Array<ArrayBuffer>. Both TextEncoder
  // output and `new Uint8Array(length)` produce that exact type.
  const sendBytes = (bytes: Uint8Array<ArrayBuffer>) => {
    const ws = inst.ws;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(bytes);
    }
  };
  const sendString = (data: string) => sendBytes(new TextEncoder().encode(data));

  // Mobile CJK IME (Korean, Japanese, Chinese) composes characters via the
  // compositionstart/end lifecycle on the underlying <textarea>. xterm's
  // onData fires for every input event, which on mobile leaks intermediate
  // jamo (e.g. ㅎㅏㄴㄱㅡㄹ instead of 한글). We:
  //   1. Suppress onData while inst.composing is true.
  //   2. Send the final composed string from compositionend's event.data.
  //   3. Briefly suppress onData after compositionend, since some browsers
  //      fire xterm's input event with the same composed text right after
  //      and we'd double-send.
  const ta = (inst.terminal as unknown as { textarea?: HTMLTextAreaElement }).textarea;
  if (ta) {
    ta.addEventListener('compositionstart', () => {
      inst.composing = true;
    });
    ta.addEventListener('compositionend', (e: CompositionEvent) => {
      inst.composing = false;
      if (e.data) sendString(e.data);
      inst.compositionSuppressUntil = performance.now() + 50;
    });
  }

  inst.terminal.onData((data) => {
    if (inst.composing) return;
    if (performance.now() < inst.compositionSuppressUntil) return;
    sendString(data);
  });
  inst.terminal.attachCustomKeyEventHandler(createShiftEnterHandler(sendString));
  inst.terminal.onBinary((data) => {
    const buf = new Uint8Array(data.length);
    for (let i = 0; i < data.length; i++) buf[i] = data.charCodeAt(i);
    sendBytes(buf);
  });
}

interface WebSocketTicketResponse {
  ticket: string;
}

function isWebSocketTicketResponse(value: unknown): value is WebSocketTicketResponse {
  return typeof value === 'object' && value !== null && 'ticket' in value && typeof value.ticket === 'string';
}

async function requestWebSocketTicket(): Promise<string> {
  const res = await fetch('/api/v1/ws-ticket', {
    method: 'POST',
    credentials: 'same-origin',
  });
  if (!res.ok) {
    throw new Error(`failed to request WebSocket ticket: ${res.status}`);
  }

  const data: unknown = await res.json();
  if (!isWebSocketTicketResponse(data)) {
    throw new Error('invalid WebSocket ticket response');
  }
  return data.ticket;
}

function scheduleReconnect(inst: TerminalInstance, sid: string): void {
  clearTimeout(inst.reconnectTimer);
  const delay = nextReconnectDelay(inst.reconnectAttempts);
  inst.reconnectAttempts++;
  inst.reconnectTimer = window.setTimeout(() => {
    if (instances.has(sid)) {
      void connectWebSocket(inst, sid);
    }
  }, delay);
}

async function connectWebSocket(inst: TerminalInstance, sid: string): Promise<void> {
  if (inst.connecting || inst.ws?.readyState === WebSocket.CONNECTING || inst.ws?.readyState === WebSocket.OPEN) {
    return;
  }

  inst.connecting = true;

  let ticket: string;
  try {
    ticket = await requestWebSocketTicket();
  } catch (err) {
    inst.connecting = false;
    console.warn(err);
    scheduleReconnect(inst, sid);
    return;
  }

  if (!instances.has(sid)) {
    inst.connecting = false;
    return;
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const ws = new WebSocket(`${protocol}//${window.location.host}/api/v1/ws/terminal/${sid}?ticket=${encodeURIComponent(ticket)}`);
  ws.binaryType = 'arraybuffer';

  ws.onopen = () => {
    inst.connecting = false;
    inst.reconnectAttempts = 0;
    inst.attached = false;
    inst.fitAddon.fit();
  };

  ws.onmessage = (e) => {
    if (e.data instanceof ArrayBuffer) {
      inst.terminal.write(new Uint8Array(e.data));
      // getState() — never subscribe; output frames must not trigger renders.
      useAppStore.getState().markSessionOutput(sid);
    } else {
      try {
        const msg = JSON.parse(e.data);
        if (msg.type === 'attached') {
          inst.attached = true;
          const dims = inst.fitAddon.proposeDimensions();
          if (dims && dims.cols > 0 && dims.rows > 0 && inst.ws?.readyState === WebSocket.OPEN) {
            inst.lastSentCols = dims.cols;
            inst.lastSentRows = dims.rows;
            inst.ws.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }));
          }
          if (msg.recreated) {
            // Backend had to spawn a brand-new tmux session — prior shell is
            // gone. Surface this inline so the user doesn't think they just
            // got a silent reset. The dim ANSI ensures it doesn't look like
            // shell output.
            inst.terminal.writeln('\x1b[2m[mansio] previous tmux session was lost; started fresh\x1b[0m');
          }
        }
      } catch {
        // ignore non-JSON text frames
      }
    }
  };

  ws.onclose = () => {
    inst.connecting = false;
    scheduleReconnect(inst, sid);
  };

  ws.onerror = () => {
    inst.connecting = false;
  };

  inst.ws = ws;
}

// Detach the cached terminal element for `sessionId` from `container` if it
// is currently mounted there. The instance itself stays alive in the map;
// only its DOM presence is removed.
function detachFromContainer(sessionId: string, container: HTMLElement): void {
  const inst = instances.get(sessionId);
  const el = inst?.terminal.element;
  if (el && el.parentElement === container) {
    container.removeChild(el);
  }
}

export function useTerminal({ sessionId, containerRef, theme }: UseTerminalOptions) {
  const activeTheme = theme ?? darkTerminalTheme;
  const previousSessionIdRef = useRef<string | null>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const previousSessionId = previousSessionIdRef.current;
    if (previousSessionId && previousSessionId !== sessionId) {
      detachFromContainer(previousSessionId, container);
    }

    if (!sessionId) {
      previousSessionIdRef.current = null;
      return;
    }

    let inst = instances.get(sessionId);
    if (!inst) {
      inst = createInstance(activeTheme);
      instances.set(sessionId, inst);
    }

    let didReattach = false;
    if (inst.terminal.element) {
      // Re-attach the cached element to the (possibly new) container.
      // VS Code's pattern: keep one xterm instance per session alive but
      // move its DOM node in/out as the user switches tabs. Avoids
      // xterm.js's display:none corruption (xtermjs/xterm.js#494, #3029).
      if (inst.terminal.element.parentElement !== container) {
        container.appendChild(inst.terminal.element);
        didReattach = true;
      }
    } else {
      inst.terminal.open(container);
      loadWebgl(inst);
      configureTextarea(inst);
      bindInputListeners(inst);
    }

    inst.fitAddon.fit();

    if (didReattach) {
      // xterm.js doesn't auto-repaint after a DOM move, and a same-size fit() is a no-op.
      inst.webgl?.clearTextureAtlas();
      inst.terminal.refresh(0, inst.terminal.rows - 1);
    }

    if (!inst.ws || inst.ws.readyState === WebSocket.CLOSED) {
      void connectWebSocket(inst, sessionId);
    }

    previousSessionIdRef.current = sessionId;

    const resizeObserver = new ResizeObserver(() => {
      const i = instances.get(sessionId);
      if (!i) return;
      const dims = i.fitAddon.proposeDimensions();
      if (!dims || dims.cols === 0 || dims.rows === 0) return;
      if (dims.cols === i.lastSentCols && dims.rows === i.lastSentRows) return;

      clearTimeout(i.resizeDebounceTimer);
      i.resizeDebounceTimer = window.setTimeout(() => {
        const inst = instances.get(sessionId);
        if (!inst) return;
        const d = inst.fitAddon.proposeDimensions();
        if (!d || d.cols === 0 || d.rows === 0) return;
        if (d.cols === inst.lastSentCols && d.rows === inst.lastSentRows) return;
        if (!inst.attached) return;

        inst.fitAddon.fit();
        if (inst.ws?.readyState === WebSocket.OPEN) {
          inst.lastSentCols = d.cols;
          inst.lastSentRows = d.rows;
          inst.ws.send(JSON.stringify({ type: 'resize', cols: d.cols, rows: d.rows }));
        }
      }, 80);
    });
    resizeObserver.observe(container);

    return () => {
      resizeObserver.disconnect();
      clearTimeout(inst.resizeDebounceTimer);
    };
  }, [sessionId, containerRef, activeTheme]);

  useEffect(() => {
    if (!sessionId) return;
    const inst = instances.get(sessionId);
    if (!inst) return;
    inst.terminal.options.theme = activeTheme;
  }, [sessionId, activeTheme]);
}

export function disposeTerminal(sessionId: string): void {
  const inst = instances.get(sessionId);
  if (!inst) return;
  clearTimeout(inst.reconnectTimer);
  clearTimeout(inst.resizeDebounceTimer);
  inst.ws?.close();
  inst.terminal.dispose();
  instances.delete(sessionId);
}

export function pasteToTerminal(sessionId: string, text: string): boolean {
  const inst = instances.get(sessionId);
  if (!inst) return false;
  inst.terminal.paste(text);
  return true;
}

// Send raw bytes straight to the WebSocket, bypassing xterm's input pipeline.
// Used by the mobile input bar: a normal <textarea> has reliable IME handling,
// so we let the user compose text there and forward the result here. Returns
// false if the session has no open socket.
export function sendToTerminal(sessionId: string, data: string): boolean {
  const inst = instances.get(sessionId);
  if (!inst || !inst.ws || inst.ws.readyState !== WebSocket.OPEN) return false;
  inst.ws.send(new TextEncoder().encode(data));
  return true;
}
