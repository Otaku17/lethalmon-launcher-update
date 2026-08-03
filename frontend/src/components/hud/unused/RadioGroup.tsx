import { useState } from 'react';
import './RadioGroup.css';

export interface RadioItem {
  id: string;
  label: string;
}

export interface RadioGroupProps {
  name: string;
  items: RadioItem[];
  defaultValue?: string;
  onChange?: (id: string) => void;
  ariaLabel?: string;
}

/**
 * <RadioGroup
 *   name="mode"
 *   items={[{ id:'manual', label:'Manuel' }, { id:'auto', label:'Automatique' }]}
 *   defaultValue="manual"
 *   onChange={(id) => {}}
 *   ariaLabel="Mode de pilotage"
 * />
 */
export default function RadioGroup({ name, items, defaultValue, onChange, ariaLabel }: RadioGroupProps) {
  const [value, setValue] = useState(defaultValue ?? items[0]?.id);

  function handleChange(id: string) {
    setValue(id);
    onChange?.(id);
  }

  return (
    <div className="hud-radio-group" role="radiogroup" aria-label={ariaLabel}>
      {items.map((item) => (
        <label className="hud-radio" key={item.id}>
          <input
            type="radio"
            className="hud-radio__input"
            name={name}
            checked={value === item.id}
            onChange={() => handleChange(item.id)}
          />
          <span className="hud-radio__dot" />
          <span className="hud-radio__label">{item.label}</span>
        </label>
      ))}
    </div>
  );
}
