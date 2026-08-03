import { useState, ChangeEvent } from 'react';
import './RangeSlider.css';

export interface RangeSliderProps {
  label?: string;
  min?: number;
  max?: number;
  defaultValue?: number;
  unit?: string;
  onChange?: (value: number) => void;
  /** Without the built-in label/value header, to sit next to an external label. */
  compact?: boolean;
  disabled?: boolean;
}

/**
 * <RangeSlider label="LASER INTENSITY" defaultValue={60} onChange={fn} />
 * <RangeSlider compact defaultValue={80} onChange={fn} />
 */
export default function RangeSlider({
  label = 'VALEUR',
  min = 0,
  max = 100,
  defaultValue = 50,
  unit = '%',
  onChange,
  compact = false,
  disabled = false,
}: RangeSliderProps) {
  const [value, setValue] = useState(defaultValue);

  function handleChange(e: ChangeEvent<HTMLInputElement>) {
    const next = Number(e.target.value);
    setValue(next);
    onChange?.(next);
  }

  return (
    <div
      className={`hud-range ${compact ? 'hud-range--compact' : ''} ${disabled ? 'is-disabled' : ''}`.trim()}
    >
      {!compact && (
        <div className="hud-range__head">
          <span>{label}</span>
          <span className="hud-range__value">{value}{unit}</span>
        </div>
      )}
      <div className="hud-range__row">
        <input
          type="range"
          min={min}
          max={max}
          value={value}
          onChange={handleChange}
          disabled={disabled}
        />
        {compact && <span className="hud-range__value hud-range__value--inline">{value}{unit}</span>}
      </div>
    </div>
  );
}
