import { useRef, useState, useCallback, MouseEvent as ReactMouseEvent, TouchEvent as ReactTouchEvent } from 'react';
import './SlideConfirm.css';

export interface SlideConfirmProps {
  label?: string;
  engagedLabel?: string;
  onConfirm?: () => void;
}

type PointerEventLike = ReactMouseEvent | ReactTouchEvent;

function getClientX(e: PointerEventLike): number {
  return 'touches' in e && e.touches.length ? e.touches[0].clientX : (e as ReactMouseEvent).clientX;
}

/**
 * <SlideConfirm label="SLIDE TO ENGAGE" engagedLabel="ENGAGED" onConfirm={fn} />
 */
export default function SlideConfirm({
  label = 'SLIDE TO ENGAGE',
  engagedLabel = 'ENGAGED',
  onConfirm,
}: SlideConfirmProps) {
  const trackRef = useRef<HTMLDivElement>(null);
  const handleRef = useRef<HTMLDivElement>(null);
  const [engaged, setEngaged] = useState(false);
  const [pos, setPos] = useState(0);
  const [dragging, setDragging] = useState(false);
  const dragData = useRef({ startX: 0, handleStart: 0, maxTravel: 0 });

  const reset = useCallback(() => {
    setEngaged(false);
    setPos(0);
  }, []);

  function pointerDown(e: PointerEventLike) {
    if (engaged || !trackRef.current || !handleRef.current) return;
    setDragging(true);
    dragData.current.startX = getClientX(e);
    dragData.current.handleStart = pos;
    dragData.current.maxTravel = trackRef.current.clientWidth - handleRef.current.clientWidth - 6;
  }

  function pointerMove(e: PointerEventLike) {
    if (!dragging) return;
    const delta = getClientX(e) - dragData.current.startX;
    const next = Math.min(Math.max(dragData.current.handleStart + delta, 0), dragData.current.maxTravel);
    setPos(next);
  }

  function pointerUp() {
    if (!dragging) return;
    setDragging(false);
    const { maxTravel } = dragData.current;
    if (pos >= maxTravel * 0.85) {
      setPos(maxTravel);
      setEngaged(true);
      onConfirm?.();
    } else {
      setPos(0);
    }
  }

  const fillPercent = dragData.current.maxTravel
    ? (pos / dragData.current.maxTravel) * 100
    : 0;

  return (
    <div
      className={`slide-confirm ${engaged ? 'engaged' : ''}`}
      onDoubleClick={reset}
      onMouseMove={pointerMove}
      onMouseUp={pointerUp}
      onMouseLeave={pointerUp}
      onTouchMove={pointerMove}
      onTouchEnd={pointerUp}
    >
      <div className="slide-confirm__track" ref={trackRef}>
        <div
          className="slide-confirm__fill"
          style={{ width: engaged ? '100%' : `${fillPercent}%` }}
        />
        <span className="slide-confirm__label">{engaged ? engagedLabel : label}</span>
        <div
          className="slide-confirm__handle"
          ref={handleRef}
          style={{ transform: `translateX(${engaged ? dragData.current.maxTravel : pos}px)` }}
          onMouseDown={pointerDown}
          onTouchStart={pointerDown}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="square">
            <path d="m9 6 6 6-6 6" />
          </svg>
        </div>
      </div>
    </div>
  );
}
