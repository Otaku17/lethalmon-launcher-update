import { useEffect, useState } from 'react';
import { Environment, WindowMinimise, Quit } from '../../../wailsjs/runtime/runtime';
import './WindowControls.css';

const MinimiseIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="square">
    <path d="M5 12h14" />
  </svg>
);

const CloseIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="square">
    <path d="M18 6 6 18M6 6l12 12" />
  </svg>
);

const noDrag = { '--wails-draggable': 'no-drag' } as React.CSSProperties;

/**
 * Window control bar (minimize / close), to place in the title area
 * (drag-region). The window is `Frameless` + fixed size: no maximize/
 * restore button.
 *
 * <WindowControls />
 */
export default function WindowControls() {
  const [platform, setPlatform] = useState('');

  useEffect(() => {
    Environment().then((env) => setPlatform(env.platform)).catch(() => {});
  }, []);

  if (platform === 'darwin') {
    return (
      <div className="hud-wincontrols hud-wincontrols--mac" style={noDrag}>
        <button
          type="button"
          className="hud-wincontrols__dot hud-wincontrols__dot--close"
          onClick={() => Quit()}
          aria-label="Fermer"
        />
        <button
          type="button"
          className="hud-wincontrols__dot hud-wincontrols__dot--min"
          onClick={() => WindowMinimise()}
          aria-label="Réduire"
        />
      </div>
    );
  }

  return (
    <div className="hud-wincontrols" style={noDrag}>
      <button
        type="button"
        className="hud-wincontrols__btn"
        onClick={() => WindowMinimise()}
        aria-label="Réduire"
      >
        <MinimiseIcon />
      </button>
      <button
        type="button"
        className="hud-wincontrols__btn hud-wincontrols__btn--close"
        onClick={() => Quit()}
        aria-label="Fermer"
      >
        <CloseIcon />
      </button>
    </div>
  );
}
