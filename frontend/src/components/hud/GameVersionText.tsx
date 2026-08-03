import { useTranslation } from 'react-i18next';
import { useAppStatus } from '../../lib/appStatus';
import './GameVersionText.css';

/**
 * Game version / install status, as plain text (no badge).
 * <GameVersionText />
 */
export default function GameVersionText() {
  const { t } = useTranslation();
  const { installed, gameVersion } = useAppStatus();

  return (
    <div className={`hud-game-version-text ${installed ? '' : 'is-warn'}`}>
      {installed ? `${t('home.gameVersionLabel')} ${gameVersion}` : t('home.notInstalled')}
    </div>
  );
}
