import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import {
  GetGameVersion,
  GetGameVersionCheck,
  GetLauncherVersion,
  GetLauncherVersionCheck,
  IsGameRunning,
} from '../../wailsjs/go/backend/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

export interface AppStatus {
  /** Always true during the very first check on startup. */
  loading: boolean;
  /** false if the version check (GitHub) failed, e.g. no connection. */
  online: boolean;
  installed: boolean;
  gameVersion: string | null;
  latestVersion: string | null;
  updateAvailable: boolean;
  downloadUrl: string | null;
  /** true if a game instance is currently running. Shared across pages
   *  (Home, Settings, ...) to stay consistent everywhere. */
  running: boolean;
  /** Local optimistic update (e.g. right after LaunchGame/StopGame), ahead
   *  of confirmation from the actual backend process check. */
  setRunning: (running: boolean) => void;
  /** Re-runs the check (e.g. after a download/an update). */
  refresh: () => void;

  /** The launcher's own version (frontend/package.json), always known even
   *  offline. */
  launcherVersion: string | null;
  launcherLatestVersion: string | null;
  launcherUpdateAvailable: boolean;
  launcherDownloadUrl: string | null;
  /** false if the distribution server is unreachable (no connection). */
  launcherOnline: boolean;
  refreshLauncher: () => void;
}

const defaultStatus: Omit<AppStatus, 'refresh' | 'setRunning' | 'running' | 'refreshLauncher' | keyof LauncherStatus> = {
  loading: true,
  online: true,
  installed: false,
  gameVersion: null,
  latestVersion: null,
  updateAvailable: false,
  downloadUrl: null,
};

interface LauncherStatus {
  launcherVersion: string | null;
  launcherLatestVersion: string | null;
  launcherUpdateAvailable: boolean;
  launcherDownloadUrl: string | null;
  launcherOnline: boolean;
}

const defaultLauncherStatus: LauncherStatus = {
  launcherVersion: null,
  launcherLatestVersion: null,
  launcherUpdateAvailable: false,
  launcherDownloadUrl: null,
  launcherOnline: true,
};

const AppStatusContext = createContext<AppStatus>({
  ...defaultStatus,
  ...defaultLauncherStatus,
  running: false,
  refresh: () => {},
  setRunning: () => {},
  refreshLauncher: () => {},
});

// AppStatusProvider checks game and launcher version/install status once on
// mount (and on demand via refresh/refreshLauncher), and tracks whether the
// game is currently running. Mount it once near the app root.
export function AppStatusProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState(defaultStatus);
  const [running, setRunning] = useState(false);
  const [launcherStatus, setLauncherStatus] = useState(defaultLauncherStatus);

  const check = useCallback(async () => {
    let localVersion: string | null = null;

    try {
      const version = await GetGameVersion();
      localVersion = version || null;
    } catch {
      // ignore: no version.txt means not installed
    }

    try {
      const result = await GetGameVersionCheck();
      setStatus({
        loading: false,
        online: true,
        installed: Boolean(result.currentVersion || localVersion),
        gameVersion: result.currentVersion || localVersion,
        latestVersion: result.latestVersion || null,
        updateAvailable: result.updateAvailable,
        downloadUrl: result.downloadUrl || null,
      });
    } catch {
      // No connection (or GitHub unreachable): fall back to what we know
      // locally, without blocking the user if the game is already here.
      setStatus({
        loading: false,
        online: false,
        installed: Boolean(localVersion),
        gameVersion: localVersion,
        latestVersion: null,
        updateAvailable: false,
        downloadUrl: null,
      });
    }
  }, []);

  const checkLauncher = useCallback(async () => {
    let localVersion: string | null = null;

    try {
      localVersion = (await GetLauncherVersion()) || null;
    } catch {
      // ignore
    }

    try {
      const result = await GetLauncherVersionCheck();
      setLauncherStatus({
        launcherVersion: result.currentVersion || localVersion,
        launcherLatestVersion: result.latestVersion || null,
        launcherUpdateAvailable: result.updateAvailable,
        launcherDownloadUrl: result.downloadUrl || null,
        launcherOnline: true,
      });
    } catch {
      // Distribution server unreachable (no connection): no update
      // notification, but we still know the local version (it's embedded
      // in the binary, no network needed).
      setLauncherStatus({
        launcherVersion: localVersion,
        launcherLatestVersion: null,
        launcherUpdateAvailable: false,
        launcherDownloadUrl: null,
        launcherOnline: false,
      });
    }
  }, []);

  // A single check, on app mount (so on launcher startup) — not every time
  // the user navigates back to the home page.
  useEffect(() => {
    check();
    checkLauncher();
  }, [check, checkLauncher]);

  useEffect(() => {
    IsGameRunning()
      .then(setRunning)
      .catch(() => {});
    return EventsOn('game:exited', () => setRunning(false));
  }, []);

  const value = useMemo<AppStatus>(
    () => ({
      ...status,
      ...launcherStatus,
      running,
      setRunning,
      refresh: check,
      refreshLauncher: checkLauncher,
    }),
    [status, launcherStatus, running, check, checkLauncher],
  );

  return <AppStatusContext.Provider value={value}>{children}</AppStatusContext.Provider>;
}

export function useAppStatus(): AppStatus {
  return useContext(AppStatusContext);
}
