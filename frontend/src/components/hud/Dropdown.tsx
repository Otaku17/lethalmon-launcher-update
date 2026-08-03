import { useEffect, useRef, useState } from 'react';
import './Dropdown.css';

const ChevronIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="square">
    <path d="m6 9 6 6 6-6" />
  </svg>
);

export interface DropdownOption {
  value: string;
  label: string;
}

export interface DropdownProps {
  options: DropdownOption[];
  value: string;
  onChange: (value: string) => void;
  ariaLabel?: string;
  disabled?: boolean;
}

/**
 * Fully custom-styled HUD dropdown (the native `<select>` can't be styled
 * consistently across all platforms).
 *
 * <Dropdown
 *   options={[{ value:'fr', label:'Français' }, ...]}
 *   value={lang}
 *   onChange={setLang}
 * />
 */
export default function Dropdown({
  options,
  value,
  onChange,
  ariaLabel = 'Sélection',
  disabled = false,
}: DropdownProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const current = options.find((opt) => opt.value === value) ?? options[0];

  useEffect(() => {
    function handleOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    function handleEscape(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false);
    }
    document.addEventListener('click', handleOutside);
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('click', handleOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, []);

  return (
    <div className={`hud-dropdown ${open ? 'is-open' : ''} ${disabled ? 'is-disabled' : ''}`} ref={ref}>
      <button
        type="button"
        className="hud-dropdown__trigger"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={ariaLabel}
        disabled={disabled}
        onClick={(e) => { e.stopPropagation(); setOpen((v) => !v); }}
      >
        <span>{current?.label}</span>
        <ChevronIcon />
      </button>
      <div className="hud-dropdown__list" role="listbox">
        {options.map((opt) => (
          <button
            key={opt.value}
            type="button"
            role="option"
            aria-selected={opt.value === value}
            className={`hud-dropdown__item ${opt.value === value ? 'is-selected' : ''}`}
            onClick={() => { onChange(opt.value); setOpen(false); }}
          >
            {opt.label}
          </button>
        ))}
      </div>
    </div>
  );
}
