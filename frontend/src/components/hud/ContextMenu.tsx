import { useEffect, useRef, useState, ReactNode } from 'react';
import Button from './Button';
import './ContextMenu.css';

const KebabIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round">
    <circle cx="12" cy="5" r="1" /><circle cx="12" cy="12" r="1" /><circle cx="12" cy="19" r="1" />
  </svg>
);

export interface ContextMenuItem {
  label?: string;
  icon?: ReactNode;
  danger?: boolean;
  divider?: boolean;
  onSelect?: () => void;
}

export interface ContextMenuProps {
  items: ContextMenuItem[];
  triggerLabel?: string;
  triggerIcon?: ReactNode;
  /** Opens the menu upward instead of downward (useful near the bottom
   *  edge of the screen). */
  openUp?: boolean;
}

/**
 * <ContextMenu
 *   triggerLabel="Ouvrir le menu"
 *   items={[
 *     { label:'Renommer', icon:<Svg/>, onSelect:fn },
 *     { divider:true },
 *     { label:'Supprimer', icon:<Svg/>, danger:true, onSelect:fn },
 *   ]}
 * />
 */
export default function ContextMenu({
  items,
  triggerLabel = 'Ouvrir le menu',
  triggerIcon,
  openUp = false,
}: ContextMenuProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

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
    <div className={`hud-menu ${open ? 'is-open' : ''} ${openUp ? 'hud-menu--up' : ''}`} ref={ref}>
      <Button
        iconOnly
        icon={triggerIcon ?? <KebabIcon />}
        aria-haspopup="true"
        aria-expanded={open}
        aria-label={triggerLabel}
        onClick={(e) => { e.stopPropagation(); setOpen((v) => !v); }}
      />
      <div className="hud-menu__list" role="menu">
        {items.map((item, idx) =>
          item.divider ? (
            <div className="hud-menu__divider" key={idx} />
          ) : (
            <button
              key={idx}
              className={`hud-menu__item ${item.danger ? 'hud-menu__item--danger' : ''}`}
              role="menuitem"
              type="button"
              onClick={() => { item.onSelect?.(); setOpen(false); }}
            >
              {item.icon}
              {item.label}
            </button>
          )
        )}
      </div>
    </div>
  );
}
