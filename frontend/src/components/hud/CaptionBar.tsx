import { ReactNode } from 'react';
import './CaptionBar.css';

export type CaptionBarVariant = 'cyan' | 'warn' | 'danger';

export interface CaptionBarProps {
  /** Text shown, usually prefixed with "//" (e.g. "// SETUP_REQUIRED"). */
  label: string;
  variant?: CaptionBarVariant;
  /** Optional content aligned to the right (status, counter, action...). */
  children?: ReactNode;
  className?: string;
}

/**
 * <CaptionBar label="// SYSTEME_FICHIERS" />
 * <CaptionBar label="// ALERTE" variant="warn">2 EN ATTENTE</CaptionBar>
 */
export default function CaptionBar({ label, variant = 'cyan', children, className = '' }: CaptionBarProps) {
  return (
    <div className={`hud-captionbar hud-captionbar--${variant} ${className}`.trim()}>
      <span className="hud-captionbar__label">{label}</span>
      <span className="hud-captionbar__rule" />
      {children && <span className="hud-captionbar__extra">{children}</span>}
    </div>
  );
}
