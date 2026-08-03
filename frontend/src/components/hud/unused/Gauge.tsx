import './Gauge.css';

const RADIUS = 40;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS; // ≈ 251.2

export interface GaugeProps {
  value?: number;
  metaLabel?: string;
  metaValue?: string;
}

/**
 * <Gauge value={65} metaLabel="BOUCLIER AVANT" metaValue="NOMINAL" />
 */
export default function Gauge({ value = 0, metaLabel, metaValue }: GaugeProps) {
  const clamped = Math.min(100, Math.max(0, value));
  const offset = CIRCUMFERENCE - (CIRCUMFERENCE * clamped) / 100;

  return (
    <div className="hud-gauge">
      <div className="hud-gauge__figure">
        <svg viewBox="0 0 96 96">
          <circle className="hud-gauge__track" cx={48} cy={48} r={RADIUS} />
          <circle
            className="hud-gauge__fill"
            cx={48} cy={48} r={RADIUS}
            strokeDasharray={CIRCUMFERENCE}
            strokeDashoffset={offset}
          />
        </svg>
        <div className="hud-gauge__readout">
          <strong>{Math.round(clamped)}</strong>
          <span>%</span>
        </div>
      </div>
      {(metaLabel || metaValue) && (
        <div className="hud-gauge__meta">
          {metaLabel && <span className="hud-gauge__meta-label">{metaLabel}</span>}
          {metaValue && <span className="hud-gauge__meta-value">{metaValue}</span>}
        </div>
      )}
    </div>
  );
}
