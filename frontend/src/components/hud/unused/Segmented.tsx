import { useState } from 'react';
import './Segmented.css';

export interface SegmentedOption {
  value: string;
  label: string;
}

export interface SegmentedProps {
  name: string;
  options: SegmentedOption[];
  defaultValue?: string;
  onChange?: (value: string) => void;
  ariaLabel?: string;
}

/**
 * <Segmented
 *   name="segview"
 *   options={[{ value:'2d', label:'2D' }, { value:'3d', label:'3D' }]}
 *   defaultValue="2d"
 *   onChange={fn}
 *   ariaLabel="Vue de la carte"
 * />
 */
export default function Segmented({ name, options, defaultValue, onChange, ariaLabel }: SegmentedProps) {
  const [value, setValue] = useState(defaultValue ?? options[0]?.value);

  function handleChange(v: string) {
    setValue(v);
    onChange?.(v);
  }

  return (
    <div className="hud-segmented" role="radiogroup" aria-label={ariaLabel}>
      {options.map((opt) => (
        <label key={opt.value}>
          <input
            type="radio"
            className="hud-segmented__input"
            name={name}
            checked={value === opt.value}
            onChange={() => handleChange(opt.value)}
          />
          <span className="hud-segmented__opt">{opt.label}</span>
        </label>
      ))}
    </div>
  );
}
