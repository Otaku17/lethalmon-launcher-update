import { ReactNode } from 'react';
import './Alert.css';

export type AlertVariant = 'info' | 'warn' | 'danger';

const icons: Record<AlertVariant, ReactNode> = {
  info: (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="square">
      <circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" />
    </svg>
  ),
  warn: (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="square">
      <path d="M12 9v4M12 17h.01M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" />
    </svg>
  ),
  danger: (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="square">
      <path d="M12 9v4M12 17h.01" /><circle cx="12" cy="12" r="10" />
    </svg>
  ),
};

export interface AlertProps {
  variant?: AlertVariant;
  title: string;
  children: ReactNode;
  /** Optional element aligned to the right (e.g. a compact action button). */
  action?: ReactNode;
}

/**
 * <Alert variant="warn" title="TEMPÉRATURE ÉLEVÉE">Réacteur au-dessus du seuil.</Alert>
 * <Alert variant="info" title="MISE À JOUR" action={<Button iconOnly icon={<Svg/>} />}>...</Alert>
 */
export function Alert({ variant = 'info', title, children, action }: AlertProps) {
  return (
    <div className={`hud-alert ${variant !== 'info' ? `hud-alert--${variant}` : ''}`}>
      {icons[variant]}
      <div className="hud-alert__body">
        <div className="hud-alert__title">{title}</div>
        <div className="hud-alert__text">{children}</div>
      </div>
      {action && <div className="hud-alert__action">{action}</div>}
    </div>
  );
}

export interface AlertItem {
  variant: AlertVariant;
  title: string;
  text: ReactNode;
}

export interface AlertListProps {
  alerts: AlertItem[];
}

/** <AlertList alerts={[{ variant:'info', title:'...', text:'...' }, ...]} /> */
export default function AlertList({ alerts }: AlertListProps) {
  return (
    <div className="hud-alerts">
      {alerts.map((a, i) => (
        <Alert key={i} variant={a.variant} title={a.title}>{a.text}</Alert>
      ))}
    </div>
  );
}
