import {
  GetGameReleases,
  GetLauncherReleases,
} from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

export type ReleaseKind = 'game' | 'launcher';

const fetchers: Record<ReleaseKind, () => Promise<main.Release[]>> = {
  game: GetGameReleases,
  launcher: GetLauncherReleases,
};

const cache: Partial<Record<ReleaseKind, Promise<main.Release[]>>> = {};

/**
 * Fetches releases for the given kind once per session, caching the
 * in-flight/resolved promise so repeated calls (e.g. navigating back to
 * the changelog page) never hit the GitHub API again.
 */
export function getReleases(kind: ReleaseKind): Promise<main.Release[]> {
  if (!cache[kind]) {
    cache[kind] = fetchers[kind]();
  }
  return cache[kind]!;
}
