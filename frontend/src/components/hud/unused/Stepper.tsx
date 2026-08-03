import { useState } from 'react';
import './Stepper.css';

export interface StepperProps {
  defaultValue?: number;
  min?: number;
  max?: number;
  unit?: string;
  onChange?: (value: number) => void;
}

/**
 * <Stepper defaultValue={4} min={0} max={99} unit="UNITÉS" onChange={fn} />
 */
export default function Stepper({ defaultValue = 0, min = 0, max = 99, unit = '', onChange }: StepperProps) {
  const [value, setValue] = useState(defaultValue);

  function update(next: number) {
    const clamped = Math.min(max, Math.max(min, next));
    setValue(clamped);
    onChange?.(clamped);
  }

  return (
    <div className="hud-stepper">
      <button className="hud-stepper__btn" type="button" aria-label="Diminuer" onClick={() => update(value - 1)}>−</button>
      <span className="hud-stepper__value">
        {value}
        {unit && <span className="hud-stepper__unit">{unit}</span>}
      </span>
      <button className="hud-stepper__btn" type="button" aria-label="Augmenter" onClick={() => update(value + 1)}>+</button>
    </div>
  );
}
