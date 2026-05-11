import { useCallback, useEffect, useRef, useState } from 'react';
import { useTerminal, pasteToTerminal } from '../../hooks/useTerminal';
import { useEffectiveTheme } from '../../hooks/useEffectiveTheme';
import { useMediaQuery, MOBILE_QUERY } from '../../hooks/useMediaQuery';
import { uploadFile } from '../../api/upload';
import '@xterm/xterm/css/xterm.css';

interface TerminalViewProps {
  sessionId: string | null;
}

export function TerminalView({ sessionId }: TerminalViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const { ui, terminalTheme } = useEffectiveTheme();
  const isMobile = useMediaQuery(MOBILE_QUERY);
  useTerminal({ sessionId, containerRef, theme: terminalTheme });

  // On mobile we route input through MobileInputBar instead of xterm's hidden
  // textarea (its 1px field breaks CJK IME composition on Android/iOS — see
  // MobileInputBar.tsx). Lock xterm's textarea read-only and keep it out of
  // the focus path so taps on the terminal don't summon the OS keyboard.
  useEffect(() => {
    const ta = containerRef.current?.querySelector<HTMLTextAreaElement>('textarea');
    if (!ta) return;
    if (isMobile) {
      ta.setAttribute('readonly', 'true');
      ta.setAttribute('tabindex', '-1');
      ta.setAttribute('inputmode', 'none');
    } else {
      ta.removeAttribute('readonly');
      ta.removeAttribute('inputmode');
      ta.setAttribute('tabindex', '0');
    }
  }, [isMobile, sessionId]);

  const [isDragging, setIsDragging] = useState(false);
  const [uploadingName, setUploadingName] = useState<string | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  const isFileDrag = (e: React.DragEvent<HTMLDivElement>) =>
    Array.from(e.dataTransfer.types).includes('Files');

  const onDragOver = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    if (!isFileDrag(e)) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = 'copy';
    setIsDragging(true);
  }, []);

  const onDragEnter = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    if (!isFileDrag(e)) return;
    setIsDragging(true);
  }, []);

  // Don't use a dragenter/dragleave depth counter — moving the cursor into the
  // inner xterm container fires dragleave on the outer wrapper BEFORE dragenter
  // on the child, briefly dropping the depth to 0 and flashing the overlay off
  // mid-drag. Instead: only hide when relatedTarget is null (cursor left the
  // window) or no longer contained by the wrapper.
  const onDragLeave = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    const next = e.relatedTarget as Node | null;
    if (next && wrapperRef.current?.contains(next)) return;
    setIsDragging(false);
  }, []);

  // Stop the browser from promoting an in-progress xterm text selection into
  // an HTML5 dragstart (which the user perceives as the selection "releasing"
  // a few pixels into the drag). File drags from outside the page still fire
  // — the types collection includes "Files" only for those.
  const onDragStart = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    if (isFileDrag(e)) return;
    e.preventDefault();
  }, []);

  const onDrop = useCallback(
    async (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      setIsDragging(false);

      if (!sessionId) return;
      const files = Array.from(e.dataTransfer.files);
      if (files.length === 0) return;

      setErrorMsg(null);
      for (const file of files) {
        setUploadingName(file.name);
        try {
          const result = await uploadFile(sessionId, file);
          pasteToTerminal(sessionId, result.path + ' ');
        } catch (err) {
          setErrorMsg(err instanceof Error ? err.message : String(err));
        }
      }
      setUploadingName(null);
    },
    [sessionId],
  );

  return (
    <div
      ref={wrapperRef}
      onDragOver={onDragOver}
      onDragEnter={onDragEnter}
      onDragLeave={onDragLeave}
      onDragStart={onDragStart}
      onDrop={onDrop}
      style={{
        position: 'relative',
        width: '100%',
        height: '100%',
        backgroundColor: ui.terminalBg,
        // Tiny breathing room on mobile — phone screens render the first
        // column flush against the bezel without it.
        paddingLeft: isMobile ? 6 : 0,
        boxSizing: 'border-box',
        userSelect: 'text',
      }}
    >
      <div
        ref={containerRef}
        style={{
          width: '100%',
          height: '100%',
        }}
      />

      {isDragging && (
        <div
          style={{
            position: 'absolute',
            inset: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            backgroundColor: ui.dropOverlayBg,
            border: `2px dashed ${ui.dropOverlayBorder}`,
            color: ui.dropOverlayText,
            fontFamily: "'JetBrains Mono', monospace",
            fontSize: 16,
            fontWeight: 600,
            pointerEvents: 'none',
            zIndex: 10,
          }}
        >
          Drop to upload
        </div>
      )}

      {uploadingName && (
        <div
          style={{
            position: 'absolute',
            bottom: 12,
            right: 12,
            padding: '6px 12px',
            backgroundColor: ui.tabActiveBg,
            border: `1px solid ${ui.accent}`,
            borderRadius: 4,
            color: ui.textPrimary,
            fontFamily: "'JetBrains Mono', monospace",
            fontSize: 12,
            zIndex: 11,
            maxWidth: '70%',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          Uploading {uploadingName}...
        </div>
      )}

      {errorMsg && (
        <div
          style={{
            position: 'absolute',
            bottom: 12,
            left: 12,
            padding: '6px 12px',
            backgroundColor: ui.tabActiveBg,
            border: `1px solid ${ui.danger}`,
            borderRadius: 4,
            color: ui.danger,
            fontFamily: "'JetBrains Mono', monospace",
            fontSize: 12,
            zIndex: 11,
            cursor: 'pointer',
            maxWidth: '70%',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
          onClick={() => setErrorMsg(null)}
          title="Click to dismiss"
        >
          Upload failed: {errorMsg}
        </div>
      )}
    </div>
  );
}
