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
import {
  FaArrowDownShortWide,
  FaArrowRotateRight,
  FaCheck,
  FaFolderOpen,
  FaPlay,
  FaRegCopy,
  FaStop,
  FaWifi,
} from 'react-icons/fa6';
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
} from '../../wailsjs/go/backend/App';
import { BrowserOpenURL, ClipboardSetText, EventsOn } from '../../wailsjs/runtime/runtime';
import ThemedImage from '../components/hud/ThemedImage';
import GameVersionText from '../components/hud/GameVersionText';
import EasterEggRunner from '../components/hud/EasterEggRunner';
import { useAppStatus } from '../lib/appStatus';
import { errorMessage } from '../lib/errors';

const MULTI_INSTANCE_KEY = 'lethalmon.multiInstance';
const EASTER_EGG_CLICKS = 7;
const EASTER_EGG_CLICK_WINDOW_MS = 600;

// Install commands shown in the "wine required" modal: shell commands, so
// left untranslated, unlike the distro labels next to them.
const WINE_INSTALL_COMMANDS = [
  { distroKey: 'debian', command: 'sudo apt install wine' },
  { distroKey: 'fedora', command: 'sudo dnf install wine' },
  { distroKey: 'arch', command: 'sudo pacman -S wine' },
] as const;

const WINE_DOWNLOAD_URL = 'https://www.winehq.org/download';

// Microsoft's permanent redirect to the latest x64 Visual C++
// Redistributable installer (2015-2022 line) — what the game's bundled Ruby
// runtime needs (VCRUNTIME140.dll, msvcp140.dll). The launcher no longer
// downloads or runs this itself (see backend/app_vcredist_windows.go); the
// player installs it, the same way they install wine themselves on Linux.
const VCREDIST_DOWNLOAD_URL = 'https://aka.ms/vs/17/release/vc_redist.x64.exe';

interface DownloadProgress {
  stage: 'downloading' | 'extracting' | 'done';
  percent: number;
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
  const [showWineModal, setShowWineModal] = useState(false);
  const [showVCRedistModal, setShowVCRedistModal] = useState(false);
  const [copiedCommand, setCopiedCommand] = useState<string | null>(null);
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

  async function handleLaunch() {
    try {
      await LaunchGame();
      if (!multiInstance) setRunning(true);
    } catch (err) {
      // wine_not_found (Linux, the game ships a Windows build) and
      // vcredist_not_found (Windows, missing Visual C++ Redistributable) are
      // the launch failures worth surfacing: either means the player can't
      // run the game at all until they install the missing piece themselves,
      // as opposed to a transient failure that's better left silent like the
      // rest of this handler treats errors.
      const message = errorMessage(err);
      if (message === 'wine_not_found') setShowWineModal(true);
      else if (message === 'vcredist_not_found') setShowVCRedistModal(true);
    }
  }

  async function handleStop() {
    await StopGame().catch(() => {});
    setRunning(false);
  }

  async function handleCopyCommand(command: string) {
    const ok = await ClipboardSetText(command).catch(() => false);
    if (!ok) return;
    setCopiedCommand(command);
    setTimeout(() => setCopiedCommand((current) => (current === command ? null : current)), 1500);
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
      setDownloadError(errorMessage(err));
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
      const path = await SelectInstallFolder(t('install.folderPickerTitle'));
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

      {showWineModal && (
        <Modal
          variant="close"
          eyebrow={t('home.wine.modalEyebrow')}
          title={t('home.wine.modalTitle')}
          open={showWineModal}
          onClose={() => setShowWineModal(false)}
        >
          <p>{t('home.wine.modalIntro')}</p>
          <ul className="wine-modal-commands">
            {WINE_INSTALL_COMMANDS.map(({ distroKey, command }) => {
              const copied = copiedCommand === command;
              return (
                <li key={distroKey}>
                  <span className="wine-modal-commands__distro">{t(`home.wine.${distroKey}`)}</span>
                  <button
                    type="button"
                    className="wine-modal-commands__command"
                    onClick={() => handleCopyCommand(command)}
                    aria-label={t(copied ? 'home.wine.copied' : 'home.wine.copy')}
                  >
                    <code>{command}</code>
                    {copied ? <FaCheck /> : <FaRegCopy />}
                  </button>
                </li>
              );
            })}
          </ul>
          <p className="modal-link-row">
            {t('home.wine.otherDistro')}{' '}
            <button type="button" className="modal-link" onClick={() => BrowserOpenURL(WINE_DOWNLOAD_URL)}>
              {t('home.wine.otherDistroLink')}
            </button>
          </p>
          <p>{t('home.wine.modalOutro')}</p>
        </Modal>
      )}

      {showVCRedistModal && (
        <Modal
          variant="close"
          eyebrow={t('home.vcRedist.modalEyebrow')}
          title={t('home.vcRedist.modalTitle')}
          open={showVCRedistModal}
          onClose={() => setShowVCRedistModal(false)}
        >
          <p>{t('home.vcRedist.modalIntro')}</p>
          <p className="modal-link-row">
            <button
              type="button"
              className="modal-link"
              onClick={() => BrowserOpenURL(VCREDIST_DOWNLOAD_URL)}
            >
              {t('home.vcRedist.download')}
            </button>
          </p>
          <p>{t('home.vcRedist.modalOutro')}</p>
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

      {showEasterEgg && <EasterEggRunner onClose={() => setShowEasterEgg(false)} />}
    </div>
  );
}

export default HomePage;
