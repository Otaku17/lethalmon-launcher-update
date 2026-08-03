import { useId, cloneElement, ReactElement } from 'react';
import './Tooltip.css';

export type TooltipPlacement = 'top' | 'bottom' | 'left' | 'right';

export interface TooltipProps {
  text: string;
  placement?: TooltipPlacement;
  children: ReactElement;
}

/**
 * <Tooltip text="Informations système" placement="right">
 *   <Button iconOnly icon={<Svg/>} aria-label="Infos" />
 * </Tooltip>
 */
export default function Tooltip({ text, placement = 'top', children }: TooltipProps) {
  const id = useId();
  return (
    <span className="hud-tooltip">
      {cloneElement(children, { 'aria-describedby': id } as Record<string, unknown>)}
      <span
        className={`hud-tooltip__bubble hud-tooltip__bubble--${placement}`}
        id={id}
        role="tooltip"
      >
        {text}
      </span>
    </span>
  );
}
