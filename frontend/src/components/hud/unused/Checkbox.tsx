import { useState, ChangeEvent, InputHTMLAttributes } from 'react';
import './Checkbox.css';

const CheckIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={3} strokeLinecap="square">
    <path d="m5 12 5 5 9-9" />
  </svg>
);

export interface CheckboxProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'checked' | 'onChange' | 'type'> {
  label: string;
  defaultChecked?: boolean;
  checked?: boolean;
  onChange?: (checked: boolean) => void;
}

/**
 * <Checkbox label="Radar" defaultChecked onChange={fn} />
 */
export function Checkbox({ label, defaultChecked = false, checked: controlled, onChange, ...rest }: CheckboxProps) {
  const [uncontrolled, setUncontrolled] = useState(defaultChecked);
  const isControlled = controlled !== undefined;
  const isChecked = isControlled ? controlled : uncontrolled;

  function handleChange(e: ChangeEvent<HTMLInputElement>) {
    if (!isControlled) setUncontrolled(e.target.checked);
    onChange?.(e.target.checked);
  }

  return (
    <label className="hud-checkbox">
      <input
        type="checkbox"
        className="hud-checkbox__input"
        checked={isChecked}
        onChange={handleChange}
        {...rest}
      />
      <span className="hud-checkbox__box"><CheckIcon /></span>
      <span className="hud-checkbox__label">{label}</span>
    </label>
  );
}

export interface CheckboxGroupItem {
  id: string;
  label: string;
  defaultChecked?: boolean;
}

export interface CheckboxGroupProps {
  items: CheckboxGroupItem[];
  onChange?: (id: string, checked: boolean) => void;
}

/**
 * <CheckboxGroup items={[{ id:'radar', label:'Radar', defaultChecked:true }, ...]} onChange={(id, checked) => {}} />
 */
export default function CheckboxGroup({ items, onChange }: CheckboxGroupProps) {
  return (
    <div className="hud-checkbox-group">
      {items.map((item) => (
        <Checkbox
          key={item.id}
          label={item.label}
          defaultChecked={item.defaultChecked}
          onChange={(checked) => onChange?.(item.id, checked)}
        />
      ))}
    </div>
  );
}
