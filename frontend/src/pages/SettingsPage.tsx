import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { useTranslation } from 'react-i18next';
import { FaUserAstronaut } from 'react-icons/fa6';
import './Pages.css';
import './SettingsPage.css';
import Panel from '../components/hud/Panel';
import Tabs from '../components/hud/Tabs';
import Dropdown from '../components/hud/Dropdown';
import Toggle from '../components/hud/Toggle';
import RangeSlider from '../components/hud/RangeSlider';
import PathInput from '../components/hud/PathInput';
import Button from '../components/hud/Button';
import Modal from '../components/hud/Modal';
import Progress from '../components/hud/Progress';
import TeamMemberModal from '../components/TeamMemberModal';
import { getTeamMembers, type TeamMember } from '../lib/team';
import { Environment, EventsOn } from '../../wailsjs/runtime/runtime';
import {
  GetGPUs,
  GetGpuPreference,
  SetGpuPreference,
  GetInstallDir,
  SelectInstallFolder,
  MoveInstallDir,
  ResetInstallPath,
  RepairGame,
  UninstallGame,
  GetGameOpts,
  SetGameOpt,
  ResetGameOptsDefaults,
  IsGameRunning,
  StopGame,
} from '../../wailsjs/go/backend/App';
import { useAppStatus } from '../lib/appStatus';

interface MigrationProgress {
  current: number;
  total: number;
  percent: number;
  file: string;
}

interface RepairProgress {
  stage: 'downloading' | 'repairing' | 'done';
  percent: number;
}

const LANGUAGES = ['fr', 'en', 'it'];
const LANGUAGE_LABELS: Record<string, string> = {
  fr: 'Français',
  en: 'English',
  it: 'Italiano',
};

const AUTO_UPDATE_KEY = 'lethalmon.autoUpdate';
const RESOLUTION_SCALES = [1, 2, 3, 4, 5, 6];
const MULTI_INSTANCE_KEY = 'lethalmon.multiInstance';

function TeamGrid({ members, onSelect }: { members: TeamMember[]; onSelect: (member: TeamMember) => void }) {
  return (
    <div className="settings-team">
      {members.map((member) => (
        <button
          key={member.id}
          type="button"
          className="settings-team-card"
          onClick={() => onSelect(member)}
        >
          {member.image ? (
            <img
              className="settings-team-card__avatar"
              src={member.image}
              alt={member.name}
            />
          ) : (
            <div className="settings-team-card__avatar settings-team-card__avatar--placeholder">
              <FaUserAstronaut size={22} />
            </div>
          )}
          <div className="settings-team-card__name">{member.name}</div>
        </button>
      ))}
    </div>
  );
}

function SettingsPage() {
  const { t, i18n } = useTranslation();
  const {
    installed,
    running,
    setRunning,
    refresh: refreshAppStatus,
    launcherVersion,
    launcherLatestVersion,
  } = useAppStatus();
  const team = getTeamMembers();
  const coreTeam = team.filter((member) => member.group === 'team');
  const contributors = team.filter((member) => member.group === 'contributor');
  const [selected, setSelected] = useState<TeamMember | null>(null);
  const [platform, setPlatform] = useState<string | null>(null);
  const [autoUpdate, setAutoUpdate] = useState(() => localStorage.getItem(AUTO_UPDATE_KEY) === 'true');
  const [gpus, setGpus] = useState<string[] | null>(null);
  const [forceGpu, setForceGpu] = useState(false);
  const [resolutionScale, setResolutionScale] = useState('2');
  const [gameLanguage, setGameLanguage] = useState('fr');
  const [volume, setVolume] = useState(50);
  const [autoSave, setAutoSave] = useState(false);
  const [gameOptsLoaded, setGameOptsLoaded] = useState(false);
  const [multiInstance, setMultiInstance] = useState(
    () => localStorage.getItem(MULTI_INSTANCE_KEY) === 'true',
  );
  const [installPath, setInstallPath] = useState('');
  const [pendingInstallPath, setPendingInstallPath] = useState<string | null>(null);
  const [migrating, setMigrating] = useState(false);
  const [migrationProgress, setMigrationProgress] = useState<MigrationProgress | null>(null);
  const [migrationError, setMigrationError] = useState<string | null>(null);
  const [installPathError, setInstallPathError] = useState<string | null>(null);
  const [showUninstallConfirm, setShowUninstallConfirm] = useState(false);
  const [showUninstallKillConfirm, setShowUninstallKillConfirm] = useState(false);
  const [showRepairConfirm, setShowRepairConfirm] = useState(false);
  const [showRepairKillConfirm, setShowRepairKillConfirm] = useState(false);
  const [repairing, setRepairing] = useState(false);
  const [repairProgress, setRepairProgress] = useState<RepairProgress | null>(null);
  const [repairError, setRepairError] = useState<string | null>(null);
  const [uninstalling, setUninstalling] = useState(false);

  useEffect(() => {
    Environment()
      .then((env) => setPlatform(env.platform))
      .catch(() => {});
    GetGPUs()
      .then(setGpus)
      .catch(() => setGpus([]));
    GetGpuPreference()
      .then(setForceGpu)
      .catch(() => {});
    GetInstallDir()
      .then(setInstallPath)
      .catch(() => {});
  }, []);

  useEffect(() => {
    // The game must be installed to have a .gameopts to read/write — otherwise
    // we just show the default values without touching anything.
    if (!installed) return;
    GetGameOpts()
      .then((opts) => {
        if (opts.scale) setResolutionScale(opts.scale);
        if (opts.lang) setGameLanguage(opts.lang);
        if (opts.master_volume) setVolume(Number(opts.master_volume));
        if (opts.auto_save) setAutoSave(opts.auto_save === 'true');
      })
      .catch(() => {})
      .finally(() => setGameOptsLoaded(true));
  }, [installed]);

  useEffect(() => {
    const unsubscribe = EventsOn('install:migration-progress', (data: MigrationProgress) => {
      setMigrationProgress(data);
    });
    return unsubscribe;
  }, []);

  useEffect(() => {
    const unsubscribe = EventsOn('install:download-progress', (data: RepairProgress) => {
      setRepairProgress(data);
    });
    return unsubscribe;
  }, []);

  function handleAutoUpdateChange(checked: boolean) {
    setAutoUpdate(checked);
    localStorage.setItem(AUTO_UPDATE_KEY, String(checked));
  }

  function handleForceGpuChange(checked: boolean) {
    setForceGpu(checked);
    SetGpuPreference(checked).catch(() => {});
  }

  function handleResolutionScaleChange(value: string) {
    setResolutionScale(value);
    SetGameOpt('scale', value).catch(() => {});
  }

  function handleGameLanguageChange(value: string) {
    setGameLanguage(value);
    SetGameOpt('lang', value).catch(() => {});
  }

  function handleVolumeChange(value: number) {
    setVolume(value);
    SetGameOpt('master_volume', String(value)).catch(() => {});
  }

  function handleAutoSaveChange(checked: boolean) {
    setAutoSave(checked);
    SetGameOpt('auto_save', String(checked)).catch(() => {});
  }

  function handleMultiInstanceChange(checked: boolean) {
    setMultiInstance(checked);
    localStorage.setItem(MULTI_INSTANCE_KEY, String(checked));
  }

  function handleResetGeneral() {
    i18n.changeLanguage('fr');
    handleAutoUpdateChange(false);
    handleForceGpuChange(false);
    handleMultiInstanceChange(false);
  }

  async function handleResetGameOpts() {
    const opts = await ResetGameOptsDefaults().catch(() => null);
    if (!opts) return;
    if (opts.scale) setResolutionScale(opts.scale);
    if (opts.lang) setGameLanguage(opts.lang);
    if (opts.master_volume) setVolume(Number(opts.master_volume));
    if (opts.auto_save) setAutoSave(opts.auto_save === 'true');
    // Force RangeSlider (uncontrolled) to remount so it picks up the default
    // volume instead of staying on its previous internal value.
    setGameOptsLoaded((loaded) => !loaded);
  }

  async function handleBrowseInstallPath() {
    try {
      const path = await SelectInstallFolder(t('install.folderPickerTitle'));
      if (path && path !== installPath) setPendingInstallPath(path);
      setInstallPathError(null);
    } catch {
      setInstallPathError(t('install.oneDriveError'));
    }
  }

  async function handleResetInstallPath() {
    setInstallPathError(null);
    setMigrationError(null);
    setPendingInstallPath(null);
    try {
      const path = await ResetInstallPath();
      setInstallPath(path);
      refreshAppStatus();
    } catch (err) {
      setInstallPathError(String(err));
    }
  }

  async function handleConfirmMoveInstall() {
    if (!pendingInstallPath) return;
    setMigrating(true);
    setMigrationProgress(null);
    setMigrationError(null);
    const startedAt = Date.now();
    const MIN_VISIBLE_MS = 600;
    try {
      await MoveInstallDir(pendingInstallPath);
      setInstallPath(pendingInstallPath);
      setPendingInstallPath(null);
      refreshAppStatus();
    } catch (err) {
      setMigrationError(String(err));
    } finally {
      const elapsed = Date.now() - startedAt;
      if (elapsed < MIN_VISIBLE_MS) {
        await new Promise((resolve) => setTimeout(resolve, MIN_VISIBLE_MS - elapsed));
      }
      setMigrating(false);
    }
  }

  async function handleUninstallClick() {
    const isRunning = await IsGameRunning().catch(() => false);
    if (isRunning) {
      setShowUninstallKillConfirm(true);
      return;
    }
    setShowUninstallConfirm(true);
  }

  async function handleConfirmKillAndUninstall() {
    setShowUninstallKillConfirm(false);
    await StopGame().catch(() => {});
    setRunning(false);
    setShowUninstallConfirm(true);
  }

  async function handleConfirmUninstall() {
    setUninstalling(true);
    try {
      await UninstallGame();
      refreshAppStatus();
    } finally {
      setUninstalling(false);
      setShowUninstallConfirm(false);
    }
  }

  async function handleRepairClick() {
    const isRunning = await IsGameRunning().catch(() => false);
    if (isRunning) {
      setShowRepairKillConfirm(true);
      return;
    }
    setShowRepairConfirm(true);
  }

  async function handleConfirmKillAndRepair() {
    setShowRepairKillConfirm(false);
    await StopGame().catch(() => {});
    setRunning(false);
    setShowRepairConfirm(true);
  }

  async function handleConfirmRepair() {
    setShowRepairConfirm(false);
    setRepairing(true);
    setRepairProgress(null);
    setRepairError(null);
    try {
      await RepairGame();
      refreshAppStatus();
    } catch (err) {
      setRepairError(String(err));
    } finally {
      setRepairing(false);
    }
  }

  return (
    <div className="settings-page">
      <Panel eyebrow="// PARAMETRES_SYSTEME" title={t('settings.title')} className="settings-panel">
        <Tabs
          className="settings-tabs"
          tabs={[
            {
              id: 'general',
              label: t('settings.tabs.general'),
              content: (
                <div className="settings-general">
                  <div className="settings-field">
                    <div className="settings-field__row">
                      <span className="settings-field__label">{t('settings.general.language')}</span>
                      <Dropdown
                        ariaLabel={t('settings.general.language')}
                        options={LANGUAGES.map((lng) => ({ value: lng, label: LANGUAGE_LABELS[lng] }))}
                        value={i18n.resolvedLanguage ?? i18n.language}
                        onChange={(value) => i18n.changeLanguage(value)}
                      />
                    </div>
                    <p className="settings-field__description">
                      {t('settings.general.languageDescription')}
                    </p>
                  </div>

                  <div className="settings-field">
                    <div className="settings-field__row">
                      <span className="settings-field__label">
                        {t('settings.launcher.autoUpdate')}
                      </span>
                      <Toggle checked={autoUpdate} onChange={handleAutoUpdateChange} />
                    </div>
                    <p className="settings-field__description">
                      {t('settings.launcher.autoUpdateDescription')}
                    </p>
                  </div>

                  <div className="settings-field">
                    <div className="settings-field__row">
                      <span className="settings-field__label">{t('settings.general.gpu')}</span>
                      <span className="settings-field__value">
                        {gpus === null
                          ? '…'
                          : gpus[0] ?? t('settings.general.gpuNotFound')}
                      </span>
                    </div>
                    <div className="settings-field__row">
                      <span className="settings-field__label">
                        {t('settings.general.forceGpu')}
                      </span>
                      <Toggle
                        checked={forceGpu}
                        onChange={handleForceGpuChange}
                        disabled={!gpus?.length}
                      />
                    </div>
                    <p className="settings-field__description">
                      {t('settings.general.gpuDescription')}
                    </p>
                  </div>

                  <div className="settings-field">
                    <div className="settings-field__row">
                      <span className="settings-field__label">
                        {t('settings.general.multiInstance')}
                      </span>
                      <Toggle checked={multiInstance} onChange={handleMultiInstanceChange} />
                    </div>
                    <p className="settings-field__description settings-field__description--warn">
                      {t('settings.general.multiInstanceDescription')}
                    </p>
                  </div>

                  <div className="settings-field">
                    <div className="settings-field__row">
                      <span className="settings-field__label">
                        {t('settings.general.reset')}
                      </span>
                      <Button variant="ghost" onClick={handleResetGeneral}>
                        {t('actions.reset')}
                      </Button>
                    </div>
                    <p className="settings-field__description">
                      {t('settings.general.resetDescription')}
                    </p>
                  </div>
                </div>
              ),
            },
            {
              id: 'game',
              label: t('settings.tabs.game'),
              content: (
                <div className="settings-general">
                  {!installed && (
                    <p className="settings-restart-notice">
                      {t('settings.game.notInstalledNotice')}
                    </p>
                  )}
                  {installed && running && (
                    <p className="settings-restart-notice">
                      {t('settings.game.restartRequiredNotice')}
                    </p>
                  )}
                  <div className="settings-field">
                    <div className="settings-field__row">
                      <span className="settings-field__label">
                        {t('settings.game.resolution')}
                      </span>
                      <Dropdown
                        ariaLabel={t('settings.game.resolution')}
                        options={RESOLUTION_SCALES.map((scale) => ({
                          value: String(scale),
                          label: `${320 * scale}x${240 * scale}`,
                        }))}
                        value={resolutionScale}
                        onChange={handleResolutionScaleChange}
                        disabled={!installed}
                      />
                    </div>
                    <p className="settings-field__description">
                      {t('settings.game.resolutionDescription')}
                    </p>
                  </div>

                  {/*<div className="settings-field">
                    <div className="settings-field__row">
                      <span className="settings-field__label">
                        {t('settings.game.language')}
                      </span>
                      <Dropdown
                        ariaLabel={t('settings.game.language')}
                        options={LANGUAGES.map((lng) => ({ value: lng, label: LANGUAGE_LABELS[lng] }))}
                        value={gameLanguage}
                        onChange={handleGameLanguageChange}
                        disabled={!installed}
                      />
                    </div>
                    <p className="settings-field__description">
                      {t('settings.game.languageDescription')}
                    </p>
                  </div>*/}

                  <div className="settings-field">
                    <div className="settings-field__row">
                      <span className="settings-field__label">
                        {t('settings.game.volume')}
                      </span>
                      <RangeSlider
                        key={String(gameOptsLoaded)}
                        compact
                        defaultValue={volume}
                        onChange={handleVolumeChange}
                        disabled={!installed}
                      />
                    </div>
                    <p className="settings-field__description">
                      {t('settings.game.volumeDescription')}
                    </p>
                  </div>

                  <div className="settings-field">
                    <div className="settings-field__row">
                      <span className="settings-field__label">
                        {t('settings.game.autoSave')}
                      </span>
                      <Toggle checked={autoSave} onChange={handleAutoSaveChange} disabled={!installed} />
                    </div>
                    <p className="settings-field__description">
                      {t('settings.game.autoSaveDescription')}
                    </p>
                  </div>

                  <div className="settings-field">
                    <div className="settings-field__row">
                      <span className="settings-field__label">
                        {t('settings.game.reset')}
                      </span>
                      <Button variant="ghost" onClick={handleResetGameOpts} disabled={!installed}>
                        {t('actions.reset')}
                      </Button>
                    </div>
                    <p className="settings-field__description">
                      {t('settings.game.resetDescription')}
                    </p>
                  </div>

                  <div className="settings-field">
                    <div className="settings-field__row">
                      <span className="settings-field__label">
                        {t('settings.game.installPath')}
                      </span>
                      <Button variant="ghost" onClick={handleResetInstallPath}>
                        {t('settings.game.resetInstallPath')}
                      </Button>
                    </div>
                    <PathInput
                      value={installPath}
                      placeholder={t('install.placeholder')}
                      onBrowse={handleBrowseInstallPath}
                    />
                    {pendingInstallPath && (
                      <div className="settings-install-pending">
                        <p className="settings-field__description">
                          {t('settings.game.installPathPendingMove', { path: pendingInstallPath })}
                        </p>
                        <div className="settings-install-pending__actions">
                          <Button variant="ghost" onClick={() => setPendingInstallPath(null)}>
                            {t('install.cancel')}
                          </Button>
                          <Button onClick={handleConfirmMoveInstall}>
                            {t('settings.game.movePathConfirm')}
                          </Button>
                        </div>
                      </div>
                    )}
                    {migrationError && (
                      <p className="settings-field__description settings-field__description--error">
                        {t('settings.game.installPathMoveError', { error: migrationError })}
                      </p>
                    )}
                    {installPathError && (
                      <p className="settings-field__description settings-field__description--error">
                        {installPathError}
                      </p>
                    )}
                    <p className="settings-field__description">
                      {t('settings.game.installPathDescription')}
                    </p>
                  </div>

                  <div className="settings-field">
                    <div className="settings-field__row">
                      <span className="settings-field__label">
                        {t('settings.game.repair')}
                      </span>
                      <Button
                        variant="ghost"
                        disabled={!installed}
                        onClick={handleRepairClick}
                      >
                        {t('settings.game.repairAction')}
                      </Button>
                    </div>
                    {repairError && (
                      <p className="settings-field__description settings-field__description--error">
                        {t('settings.game.repairError', { error: repairError })}
                      </p>
                    )}
                    <p className="settings-field__description">
                      {t('settings.game.repairDescription')}
                    </p>
                  </div>

                  <div className="settings-field">
                    <div className="settings-field__row">
                      <span className="settings-field__label">
                        {t('settings.game.uninstall')}
                      </span>
                      <Button
                        variant="danger"
                        className="settings-uninstall-btn"
                        disabled={!installed}
                        onClick={handleUninstallClick}
                      >
                        {t('settings.game.uninstallAction')}
                      </Button>
                    </div>
                    <p className="settings-field__description">
                      {t('settings.game.uninstallDescription')}
                    </p>
                  </div>
                </div>
              ),
            },
            {
              id: 'about',
              label: t('settings.tabs.about'),
              content: (
                <div className="settings-about">
                  <h3 className="settings-about__heading">{t('settings.about.heading')}</h3>
                  {(t('settings.about.paragraphs', { returnObjects: true }) as string[]).map(
                    (paragraph, index) => (
                      <p key={index} className="settings-about__paragraph">
                        {paragraph}
                      </p>
                    ),
                  )}

                  <h3 className="settings-about__heading settings-about__heading--team">
                    {t('settings.tabs.launcher')}
                  </h3>
                  <div className="settings-info-list">
                    <div className="settings-info-row">
                      <span className="settings-info-row__label">
                        {t('settings.launcher.currentVersion')}
                      </span>
                      <span className="settings-info-row__value">{launcherVersion ?? '—'}</span>
                    </div>
                    <div className="settings-info-row">
                      <span className="settings-info-row__label">
                        {t('settings.launcher.latestVersion')}
                      </span>
                      <span className="settings-info-row__value">
                        {launcherLatestVersion ?? '—'}
                      </span>
                    </div>
                    <div className="settings-info-row">
                      <span className="settings-info-row__label">
                        {t('settings.launcher.platform')}
                      </span>
                      <span className="settings-info-row__value">{platform ?? '—'}</span>
                    </div>
                    <div className="settings-info-row">
                      <span className="settings-info-row__label">
                        {t('settings.launcher.tech')}
                      </span>
                      <span className="settings-info-row__value">Wails · React · Go</span>
                    </div>
                  </div>

                  <h3 className="settings-about__heading settings-about__heading--team">
                    {t('settings.about.teamTitle')}
                  </h3>
                  <TeamGrid members={coreTeam} onSelect={setSelected} />

                  <h3 className="settings-about__heading settings-about__heading--team">
                    {t('settings.about.contributorsTitle')}
                  </h3>
                  <TeamGrid members={contributors} onSelect={setSelected} />
                </div>
              ),
            },
          ]}
        />
      </Panel>

      <TeamMemberModal member={selected} onClose={() => setSelected(null)} />

      {migrating &&
        createPortal(
          <div className="settings-migration-overlay">
            <div className="settings-migration-box">
              <Progress
                label={t('settings.game.migrationProgressLabel')}
                value={migrationProgress?.percent ?? 0}
              />
              {migrationProgress && (
                <p className="settings-migration-box__file">{migrationProgress.file}</p>
              )}
            </div>
          </div>,
          document.body,
        )}

      {repairing &&
        createPortal(
          <div className="settings-migration-overlay">
            <div className="settings-migration-box">
              <Progress
                label={t(`install.${repairProgress?.stage ?? 'downloading'}`)}
                value={repairProgress?.percent ?? 0}
              />
            </div>
          </div>,
          document.body,
        )}

      {showRepairKillConfirm && (
        <Modal
          eyebrow="// ATTENTION"
          title={t('settings.game.repairKillConfirmTitle')}
          open={showRepairKillConfirm}
          confirmLabel={t('settings.game.repairAction')}
          confirmVariant="danger"
          cancelLabel={t('install.cancel')}
          onCancel={() => setShowRepairKillConfirm(false)}
          onConfirm={handleConfirmKillAndRepair}
        >
          <p>{t('settings.game.repairKillConfirmDescription')}</p>
        </Modal>
      )}

      {showRepairConfirm && (
        <Modal
          eyebrow="// REPARATION"
          title={t('settings.game.repairConfirmTitle')}
          open={showRepairConfirm}
          confirmLabel={t('settings.game.repairAction')}
          cancelLabel={t('install.cancel')}
          onCancel={() => setShowRepairConfirm(false)}
          onConfirm={handleConfirmRepair}
        >
          <p>{t('settings.game.repairConfirmDescription')}</p>
        </Modal>
      )}

      {showUninstallKillConfirm && (
        <Modal
          eyebrow="// ATTENTION"
          title={t('settings.game.uninstallKillConfirmTitle')}
          open={showUninstallKillConfirm}
          confirmLabel={t('settings.game.uninstallAction')}
          confirmVariant="danger"
          cancelLabel={t('install.cancel')}
          onCancel={() => setShowUninstallKillConfirm(false)}
          onConfirm={handleConfirmKillAndUninstall}
        >
          <p>{t('settings.game.uninstallKillConfirmDescription')}</p>
        </Modal>
      )}

      {showUninstallConfirm && (
        <Modal
          eyebrow="// DESINSTALLATION"
          title={t('settings.game.uninstallConfirmTitle')}
          open={showUninstallConfirm}
          confirmLabel={uninstalling ? '…' : t('settings.game.uninstallAction')}
          confirmVariant="danger"
          cancelLabel={t('install.cancel')}
          onCancel={() => setShowUninstallConfirm(false)}
          onConfirm={handleConfirmUninstall}
        >
          <p>{t('settings.game.uninstallConfirmDescription')}</p>
        </Modal>
      )}
    </div>
  );
}

export default SettingsPage;
