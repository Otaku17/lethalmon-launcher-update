import { useState, InputHTMLAttributes, ChangeEvent } from 'react';
import './Toggle.css';

export interface ToggleProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'checked' | 'onChange' | 'type'> {
  defaultChecked?: boolean;
  checked?: boolean;
  onLabel?: string;
  offLabel?: string;
  onChange?: (checked: boolean) => void;
}

/**
 * <Toggle defaultChecked={false} onChange={fn} />
 * <Toggle defaultChecked={false} onLabel="ONLINE" offLabel="OFFLINE" onChange={fn} />
 */
export default function Toggle({
  defaultChecked = false,
  checked: controlledChecked,
  onLabel,
  offLabel,
  onChange,
  ...rest
}: ToggleProps) {
  const [uncontrolled, setUncontrolled] = useState(defaultChecked);
  const isControlled = controlledChecked !== undefined;
  const checked = isControlled ? controlledChecked : uncontrolled;
  const status = checked ? onLabel : offLabel;

  function handleChange(e: ChangeEvent<HTMLInputElement>) {
    if (!isControlled) setUncontrolled(e.target.checked);
    onChange?.(e.target.checked);
  }

  return (
    <label className="hud-toggle">
      <input
        type="checkbox"
        className="hud-toggle__input"
        checked={checked}
        onChange={handleChange}
        {...rest}
      />
      <span className="hud-toggle__track">
        <span className="hud-toggle__thumb" />
      </span>
      {status && <span className="hud-toggle__status">{status}</span>}
    </label>
  );
}
