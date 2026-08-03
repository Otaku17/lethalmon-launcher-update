import './Progress.css';

export interface ProgressProps {
  label?: string;
  value?: number;
}

/**
 * <Progress label="CHARGEMENT SYSTÈME" value={64} />
 */
export default function Progress({ label = 'PROGRESS', value = 0 }: ProgressProps) {
  const clamped = Math.min(100, Math.max(0, value));
  return (
    <div className="hud-progress">
      <div className="hud-progress__head">
        <span>{label}</span>
        <span className="hud-progress__value">{clamped}%</span>
      </div>
      <div className="hud-progress__track">
        <div className="hud-progress__fill" style={{ width: `${clamped}%` }} />
      </div>
    </div>
  );
}
