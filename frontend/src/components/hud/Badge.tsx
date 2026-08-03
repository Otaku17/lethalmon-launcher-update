import { ReactNode } from 'react';
import './Badge.css';

export type BadgeVariant = 'ok' | 'warn' | 'danger';

export interface BadgeProps {
  variant?: BadgeVariant;
  children: ReactNode;
}

/**
 * <Badge variant="ok">EN LIGNE</Badge>
 */
export function Badge({ variant = 'ok', children }: BadgeProps) {
  return <span className={`hud-badge hud-badge--${variant}`}>{children}</span>;
}

export interface BadgeGroupItem {
  variant: BadgeVariant;
  label: string;
}

export interface BadgeGroupProps {
  badges: BadgeGroupItem[];
}

/** <BadgeGroup badges={[{ variant:'ok', label:'EN LIGNE' }, ...]} /> */
export default function BadgeGroup({ badges }: BadgeGroupProps) {
  return (
    <div className="hud-badges">
      {badges.map((b, i) => (
        <Badge key={i} variant={b.variant}>{b.label}</Badge>
      ))}
    </div>
  );
}
