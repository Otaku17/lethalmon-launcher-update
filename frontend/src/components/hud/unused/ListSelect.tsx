import { useState, KeyboardEvent } from 'react';
import './ListSelect.css';

export interface ListItem {
  id: string;
  label: string;
}

export interface ListSelectProps {
  items: ListItem[];
  defaultSelectedId?: string;
  activeStatus?: string;
  idleStatus?: string;
  onSelect?: (id: string) => void;
  ariaLabel?: string;
}

/**
 * <ListSelect
 *   items={[{ id:'NAV', label:'Navigation' }, { id:'PWR', label:'Alimentation' }]}
 *   defaultSelectedId="NAV"
 *   onSelect={fn}
 * />
 */
export default function ListSelect({
  items,
  defaultSelectedId,
  activeStatus = 'ACTIF',
  idleStatus = 'EN VEILLE',
  onSelect,
  ariaLabel = 'Sélection',
}: ListSelectProps) {
  const [selectedId, setSelectedId] = useState(defaultSelectedId ?? items[0]?.id);

  function select(id: string) {
    setSelectedId(id);
    onSelect?.(id);
  }

  function handleKeyDown(e: KeyboardEvent<HTMLDivElement>, idx: number) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      select(items[idx].id);
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      document.getElementById(`hud-list-item-${items[(idx + 1) % items.length].id}`)?.focus();
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      document.getElementById(`hud-list-item-${items[(idx - 1 + items.length) % items.length].id}`)?.focus();
    }
  }

  return (
    <div className="hud-list" role="listbox" aria-label={ariaLabel}>
      {items.map((item, idx) => {
        const selected = item.id === selectedId;
        return (
          <div
            key={item.id}
            id={`hud-list-item-${item.id}`}
            className={`hud-list__item ${selected ? 'is-selected' : ''}`}
            role="option"
            aria-selected={selected}
            tabIndex={selected ? 0 : -1}
            onClick={() => select(item.id)}
            onKeyDown={(e) => handleKeyDown(e, idx)}
          >
            <span className="hud-list__id">{item.id}</span>
            <span className="hud-list__label">{item.label}</span>
            <span className="hud-list__status">{selected ? activeStatus : idleStatus}</span>
          </div>
        );
      })}
    </div>
  );
}
