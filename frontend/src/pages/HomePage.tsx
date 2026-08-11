import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { useTranslation } from 'react-i18next';
import logo from '../assets/images/logo.svg';
import RotatingTagline from '../components/RotatingTagline';
import './Pages.css';
import Modal from '../components/hud/Modal';
import Button from '../components/hud/Button';
import Progress from '../components/hud/Progress';
import ContextMenu from '../components/hud/ContextMenu';
import { FaArrowDownShortWide, FaArrowRotateRight, FaFolderOpen, FaPlay, FaStop, FaWifi } from 'react-icons/fa6';
import PathInput from '../components/hud/PathInput';
import {
  GetInstallDir,
  SelectInstallFolder,
  MoveInstallDir,
  DownloadGame,
  LaunchGame,
  StopGame,
  IsGameRunning,
  OpenInstallFolder,
} from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import ThemedImage from '../components/hud/ThemedImage';
import GameVersionText from '../components/hud/GameVersionText';
import EasterEggRunner from '../components/hud/EasterEggRunner';
import { useAppStatus } from '../lib/appStatus';

const MULTI_INSTANCE_KEY = 'lethalmon.multiInstance';
const EASTER_EGG_CLICKS = 7;
const EASTER_EGG_CLICK_WINDOW_MS = 600;

interface DownloadProgress {
  stage: 'downloading' | 'extracting' | 'done';
  percent: number;
}

interface VCRedistProgress {
  stage: 'downloading' | 'installing' | 'done' | 'failed';
  percent: number;
  error?: string;
}

function HomePage() {
  const { t } = useTranslation();
  const { installed, online, updateAvailable, running, setRunning, refresh } = useAppStatus();
  const [showInstallModal, setShowInstallModal] = useState(false);
  const [installPath, setInstallPath] = useState('');
  const [downloading, setDownloading] = useState(false);
  const [downloadProgress, setDownloadProgress] = useState<DownloadProgress | null>(null);
  const [downloadError, setDownloadError] = useState<string | null>(null);
  const [installPathError, setInstallPathError] = useState<string | null>(null);
  const [showKillConfirm, setShowKillConfirm] = useState(false);
  const [vcRedistProgress, setVcRedistProgress] = useState<VCRedistProgress | null>(null);
  const [showEasterEgg, setShowEasterEgg] = useState(false);
  const logoClicks = useRef(0);
  const lastLogoClickAt = useRef(0);
  const multiInstance = localStorage.getItem(MULTI_INSTANCE_KEY) === 'true';

  useEffect(() => {
    GetInstallDir()
      .then(setInstallPath)
      .catch(() => {});
  }, []);

  useEffect(() => {
    return EventsOn('install:download-progress', (data: DownloadProgress) => {
      setDownloadProgress(data);
    });
  }, []);

  useEffect(() => {
    return EventsOn('game:vcredist-progress', (data: VCRedistProgress) => {
      setVcRedistProgress(data);
    });
  }, []);

  async function handleLaunch() {
    setVcRedistProgress(null);
    await LaunchGame().catch(() => {});
    if (!multiInstance) setRunning(true);
  }

  async function handleStop() {
    await StopGame().catch(() => {});
    setRunning(false);
  }

  async function performDownload() {
    setDownloading(true);
    setDownloadProgress(null);
    setDownloadError(null);
    const startedAt = Date.now();
    const MIN_VISIBLE_MS = 600;

    try {
      const isRunning = await IsGameRunning().catch(() => false);
      if (isRunning) {
        await StopGame().catch(() => {});
        setRunning(false);
        // Give the OS time to release the killed process's files.
        await new Promise((resolve) => setTimeout(resolve, 500));
      }

      const currentDir = await GetInstallDir();
      if (installPath && installPath !== currentDir) {
        await MoveInstallDir(installPath);
      }
      await DownloadGame();
      setInstallPath(await GetInstallDir());
      refresh();
    } catch (err) {
      setDownloadError(String(err));
    } finally {
      const elapsed = Date.now() - startedAt;
      if (elapsed < MIN_VISIBLE_MS) {
        await new Promise((resolve) => setTimeout(resolve, MIN_VISIBLE_MS - elapsed));
      }
      setDownloading(false);
    }
  }

  async function handleConfirmInstall() {
    setShowInstallModal(false);
    await performDownload();
  }

  async function handleUpdateClick() {
    const isRunning = await IsGameRunning().catch(() => false);
    if (isRunning) {
      setShowKillConfirm(true);
      return;
    }
    await performDownload();
  }

  async function handleConfirmKillAndUpdate() {
    setShowKillConfirm(false);
    await performDownload();
  }

  async function handleBrowseInstallPath() {
    try {
      const path = await SelectInstallFolder();
      if (path) {
        setInstallPath(path);
        setInstallPathError(null);
      }
    } catch {
      setInstallPathError(t('install.oneDriveError'));
    }
  }

  function handleLogoClick() {
    const now = Date.now();
    logoClicks.current = now - lastLogoClickAt.current > EASTER_EGG_CLICK_WINDOW_MS ? 1 : logoClicks.current + 1;
    lastLogoClickAt.current = now;

    if (logoClicks.current >= EASTER_EGG_CLICKS) {
      logoClicks.current = 0;
      setShowEasterEgg(true);
    }
  }

  return (
    <div className="page">
      <GameVersionText />
      <div className="home-logo-wrap" onClick={handleLogoClick}>
        {/*<img className="home-logo" src={logo} alt="Lethalmon Launcher" />*/}
        <ThemedImage
          className="home-logo"
          src={logo}
          alt="Lethalmon Launcher"
          intensity={1}
        />
      </div>
      <RotatingTagline />
      <div className="home-actions">
        {installed && updateAvailable && (
          <Button variant="warn" icon={<FaArrowRotateRight />} onClick={handleUpdateClick}>
            {t('actions.update')}
          </Button>
        )}
        <div className="home-actions-row">
          {installed && (
            <ContextMenu
              triggerLabel={t('home.moreActions')}
              openUp
              items={[
                {
                  label: t('settings.game.openFolderAction'),
                  icon: <FaFolderOpen />,
                  onSelect: () => OpenInstallFolder(),
                },
              ]}
            />
          )}
          {installed ? (
            running && !multiInstance ? (
              <Button variant="danger" icon={<FaStop />} onClick={handleStop}>
                {t('actions.stop')}
              </Button>
            ) : (
              <Button icon={<FaPlay />} onClick={handleLaunch}>
                {t('actions.launch')}
              </Button>
            )
          ) : online ? (
            <Button
              icon={<FaArrowDownShortWide />}
              onClick={() => {
                setInstallPathError(null);
                setShowInstallModal(true);
              }}
            >
              {t('actions.download')}
            </Button>
          ) : (
            <div className="home-offline-notice">
              <FaWifi />
              {t('home.connectionRequired')}
            </div>
          )}
        </div>
      </div>
      {showInstallModal && (
        <Modal
          eyebrow={t('install.eyebrow')}
          size="medium"
          open={showInstallModal}
          title={t('install.title')}
          onCancel={() => setShowInstallModal(false)}
          onConfirm={handleConfirmInstall}
        >
          <p>{t('install.description')}</p>
          <PathInput
            value={installPath}
            placeholder={t('install.placeholder')}
            onBrowse={handleBrowseInstallPath}
          />
          {installPathError && (
            <p className="home-download-error">{installPathError}</p>
          )}
        </Modal>
      )}

      {showKillConfirm && (
        <Modal
          eyebrow="// ATTENTION"
          title={t('home.updateKillConfirmTitle')}
          open={showKillConfirm}
          confirmLabel={t('actions.update')}
          confirmVariant="danger"
          cancelLabel={t('install.cancel')}
          onCancel={() => setShowKillConfirm(false)}
          onConfirm={handleConfirmKillAndUpdate}
        >
          <p>{t('home.updateKillConfirmDescription')}</p>
        </Modal>
      )}

      {downloadError && (
        <p className="home-download-error">{downloadError}</p>
      )}

      {downloading &&
        createPortal(
          <div className="home-download-overlay">
            <div className="home-download-box">
              <Progress
                label={t(`install.${downloadProgress?.stage ?? 'downloading'}`)}
                value={downloadProgress?.percent ?? 0}
              />
            </div>
          </div>,
          document.body,
        )}

      {vcRedistProgress &&
        (vcRedistProgress.stage === 'downloading' || vcRedistProgress.stage === 'installing') &&
        createPortal(
          <div className="home-download-overlay">
            <div className="home-download-box">
              <Progress
                label={t(`home.vcRedist.${vcRedistProgress.stage}`)}
                value={vcRedistProgress.percent}
              />
            </div>
          </div>,
          document.body,
        )}

      {vcRedistProgress?.stage === 'failed' && (
        <p className="home-download-error">
          {t('home.vcRedist.failed', { error: vcRedistProgress.error })}
        </p>
      )}

      {showEasterEgg && <EasterEggRunner onClose={() => setShowEasterEgg(false)} />}
    </div>
  );
}

export default HomePage;
