import { SelectHTMLAttributes } from 'react';
import './Select.css';

const ChevronIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="square">
    <path d="m6 9 6 6 6-6" />
  </svg>
);

export interface SelectOption {
  value: string;
  label: string;
}

export interface SelectProps extends Omit<SelectHTMLAttributes<HTMLSelectElement>, 'onChange' | 'defaultValue'> {
  options: SelectOption[];
  defaultValue?: string;
  onChange?: (value: string) => void;
}

/**
 * <Select
 *   options={[{ value:'nav', label:'Navigation' }, ...]}
 *   defaultValue="nav"
 *   onChange={(value) => {}}
 * />
 */
export default function Select({ options, defaultValue, onChange, ...rest }: SelectProps) {
  return (
    <div className="hud-select">
      <select defaultValue={defaultValue} onChange={(e) => onChange?.(e.target.value)} {...rest}>
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>{opt.label}</option>
        ))}
      </select>
      <ChevronIcon />
    </div>
  );
}
