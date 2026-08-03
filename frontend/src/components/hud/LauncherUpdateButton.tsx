import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { useTranslation } from 'react-i18next';
import { FaArrowRotateRight } from 'react-icons/fa6';
import { Alert } from './Alert';
import Button from './Button';
import Progress from './Progress';
import './LauncherUpdateButton.css';
import { UpdateLauncher } from '../../../wailsjs/go/main/App';
import { EventsOn } from '../../../wailsjs/runtime/runtime';
import { useAppStatus } from '../../lib/appStatus';

const AUTO_UPDATE_KEY = 'lethalmon.autoUpdate';
const NOTICE_DURATION_MS = 5000;

const noDrag = { '--wails-draggable': 'no-drag' } as React.CSSProperties;

interface UpdateProgress {
  stage: 'downloading' | 'installing';
  percent: number;
}

type UpdateState =
  | { phase: 'idle' }
  | { phase: 'notice' }
  | { phase: 'updating'; progress: UpdateProgress | null }
  | { phase: 'error'; message: string };

/**
 * Checks whether an update to the launcher itself is available (see
 * useAppStatus -> launcherUpdateAvailable, served by the distribution
 * server, not GitHub):
 * - if "automatic updates" is on, downloads and installs it right away on
 *   startup (the app closes then relaunches itself);
 * - otherwise, shows an informational alert for 5 seconds, then a small
 *   persistent button in the title bar (to the left of the online player
 *   count) to trigger the update manually.
 * Does nothing while offline (see launcherOnline).
 */
export default function LauncherUpdateButton() {
  const { t } = useTranslation();
  const { launcherOnline, launcherUpdateAvailable, launcherLatestVersion } = useAppStatus();
  const autoUpdate = localStorage.getItem(AUTO_UPDATE_KEY) === 'true';
  const [state, setState] = useState<UpdateState>({ phase: 'idle' });
  // Guard so we only react once to an update becoming available — orthogonal
  // to `state`, which represents what's currently shown on screen.
  const reacted = useRef(false);

  useEffect(() => {
    return EventsOn('launcher:update-progress', (data: UpdateProgress) => {
      setState((prev) => (prev.phase === 'updating' ? { phase: 'updating', progress: data } : prev));
    });
  }, []);

  useEffect(() => {
    if (!launcherOnline || !launcherUpdateAvailable) return;
    if (reacted.current) return;
    reacted.current = true;

    if (autoUpdate) {
      runUpdate();
    } else {
      setState({ phase: 'notice' });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [launcherOnline, launcherUpdateAvailable, autoUpdate]);

  useEffect(() => {
    if (state.phase !== 'notice') return;
    const timer = setTimeout(() => {
      setState((prev) => (prev.phase === 'notice' ? { phase: 'idle' } : prev));
    }, NOTICE_DURATION_MS);
    return () => clearTimeout(timer);
  }, [state.phase]);

  async function runUpdate() {
    setState({ phase: 'updating', progress: null });
    try {
      await UpdateLauncher();
      // Success: the process closes itself on the Go side (see
      // wailsruntime.Quit in UpdateLauncher) then relaunches up to date.
    } catch (err) {
      setState({ phase: 'error', message: String(err) });
    }
  }

  const showButton = !autoUpdate && launcherOnline && launcherUpdateAvailable;

  return (
    <>
      {showButton && (
        <div className="titlebar-update-btn" style={noDrag}>
          <Button
            variant="warn"
            iconOnly
            icon={<FaArrowRotateRight />}
            aria-label={t('actions.update')}
            onClick={runUpdate}
          />
        </div>
      )}

      {state.phase === 'notice' &&
        createPortal(
          <div className="launcher-update-alert">
            <Alert variant="warn" title={t('home.launcherUpdate.title')}>
              {t('home.launcherUpdate.available', { version: launcherLatestVersion })}
            </Alert>
          </div>,
          document.body,
        )}

      {(state.phase === 'updating' || state.phase === 'error') &&
        createPortal(
          <div className="launcher-update-overlay">
            <div className="launcher-update-box">
              {state.phase === 'error' ? (
                <>
                  <p className="launcher-update-error">{state.message}</p>
                  <Button variant="ghost" onClick={() => setState({ phase: 'idle' })}>
                    {t('install.cancel')}
                  </Button>
                </>
              ) : (
                <Progress
                  label={t(`home.launcherUpdate.${state.progress?.stage ?? 'downloading'}`)}
                  value={state.progress?.percent ?? 0}
                />
              )}
            </div>
          </div>,
          document.body,
        )}
    </>
  );
}
