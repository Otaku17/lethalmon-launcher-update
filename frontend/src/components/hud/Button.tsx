import { ReactNode, ButtonHTMLAttributes, MouseEvent } from 'react';
import './Button.css';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  children?: ReactNode;
  icon?: ReactNode;
  iconOnly?: boolean;
  variant?: 'default' | 'danger' | 'ghost' | 'select' | 'warn';
  /** Avec `variant="select"` : marque le bouton comme l'option active. */
  selected?: boolean;
}

/**
 * <Button>EXECUTE</Button>
 * <Button icon={<Svg/>} iconOnly aria-label="Paramètres" />
 * <Button variant="danger">PURGER</Button>
 * <Button variant="select" selected={activePage === 'home'} icon={<Svg/>} iconOnly aria-label="Accueil" />
 */
export default function Button({
  children,
  icon = null,
  iconOnly = false,
  variant = 'default',
  selected = false,
  className = '',
  onClick,
  ...rest
}: ButtonProps) {
  const handleClick = (e: MouseEvent<HTMLButtonElement>) => {
    onClick?.(e);
    // Blur after a mouse click so hover/tooltip state (driven by
    // :focus-within) doesn't stay stuck until the user clicks elsewhere.
    e.currentTarget.blur();
  };

  const classes = [
    'hud-btn',
    iconOnly ? 'hud-btn--icon' : '',
    variant === 'danger' ? 'hud-btn--danger' : '',
    variant === 'ghost' ? 'hud-btn--ghost' : '',
    variant === 'select' ? 'hud-btn--select' : '',
    variant === 'select' && selected ? 'is-selected' : '',
    variant === 'warn' ? 'hud-btn--warn' : '',
    className,
  ].filter(Boolean).join(' ');

  return (
    <button
      className={classes}
      type="button"
      aria-pressed={variant === 'select' ? selected : undefined}
      onClick={handleClick}
      {...rest}
    >
      {icon}
      {children && <span>{children}</span>}
    </button>
  );
}
