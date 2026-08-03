import { ReactNode } from 'react';
import CaptionBar from './CaptionBar';
import './Panel.css';

export interface PanelProps {
  /** Usually prefixed with "//" (e.g. "// MISSION_LOG"). */
  eyebrow?: string;
  title?: string;
  /** Extra content shown to the right of the title (badge, disclaimer...). */
  headerExtra?: ReactNode;
  children: ReactNode;
  className?: string;
}

/**
 * Reusable HUD frame (same look as `Modal`, but static, for dressing up a
 * page's content rather than a dialog).
 *
 * <Panel eyebrow="// MISSION_LOG" title="CHANGELOG">
 *   ...
 * </Panel>
 */
export default function Panel({ eyebrow, title, headerExtra, children, className = '' }: PanelProps) {
  return (
    <div className={`hud-panel ${className}`.trim()}>
      {(eyebrow || title) && (
        <div className="hud-panel__header">
          {eyebrow && <CaptionBar label={eyebrow} className="hud-panel__eyebrow" />}
          {title && (
            <div className="hud-panel__title-row">
              <h2 className="hud-panel__title">{title}</h2>
              {headerExtra}
            </div>
          )}
        </div>
      )}
      <div className="hud-panel__body">{children}</div>
    </div>
  );
}
