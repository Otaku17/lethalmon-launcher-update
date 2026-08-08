<p align="center">
  <img src="frontend/src/assets/images/logo.svg" alt="Lethalmon" width="220">
</p>

<h1 align="center">Lethalmon Launcher</h1>

<p align="center">
  Official launcher for <strong>Lethalmon</strong>, a fangame.<br>
  Built with <a href="https://wails.io">Wails v2</a> (Go backend, React/TypeScript frontend).
</p>

<p align="center">
  <img src="https://img.shields.io/github/v/release/Otaku17/lethalmon-launcher-update?label=latest%20release" alt="Latest release">
  <img src="https://img.shields.io/github/actions/workflow/status/Otaku17/lethalmon-launcher-update/build.yml?branch=main&label=build" alt="Build status">
  <img src="https://img.shields.io/github/license/Otaku17/lethalmon-launcher-update" alt="License">
</p>

This repository is the source for the desktop launcher only — the app
players install to download, update, configure, and start the game. It's
fully open source (MIT) so anyone can audit exactly what it does: no
closed-source binary is shipped without a matching commit to build it
from. The game itself lives in a separate repository
([bfloo/lethalmon-launcher](https://github.com/bfloo/lethalmon-launcher)),
which this launcher tracks for game updates the same way it tracks its
own releases here for self-updates.

## Features

- Download, install, update, and launch the game, with automatic detection
  of an existing install from the old launcher.
- Choose or change the install location at any time; uninstalling clears
  the game files while preserving save data (`Saves/`, hall of fame,
  Pokédex).
- Auto-installs the Visual C++ Redistributable (VCRUNTIME140/msvcp140) on
  first launch if missing, instead of leaving players stuck on a bare
  "DLL is missing" system dialog.
- Game settings (resolution, language, volume, auto-save, and more) linked
  directly to the game's own `.gameopts` file, so the launcher and the
  game are always in sync.
- Graphics card detection, with an option to force the dedicated GPU on
  dual-GPU setups (via Windows' per-app graphics preference).
- Self-updating, with cryptographic (ed25519) signature verification of
  update artifacts before install — an update is refused outright if it
  isn't signed with the key shipped in the launcher.
- Built-in changelog (pulled from GitHub releases), a live online player
  count (via a WebSocket connection to the game's presence server), a
  team page, and a multi-language interface (French, English, Italian).
- A hidden dino-run-style mini-game, unlocked from the home page for fun.

## How it works

The launcher is a [Wails v2](https://wails.io) desktop app (Go backend,
React/TypeScript frontend) — not a browser wrapper. Network access is
limited to a small, fixed set of destinations:

- **GitHub Releases** (`api.github.com`, `github.com`) — fetches the game's
  and launcher's release lists/changelogs, and downloads game/launcher
  update artifacts as plain `.zip`/`.exe` assets attached to public
  releases.
- **`aka.ms`** — Microsoft's own permanent redirect used to fetch the
  official Visual C++ Redistributable installer when it's missing; the
  downloaded installer's Authenticode signature is verified (must be
  validly signed by "Microsoft Corporation") before it's ever executed.
- **`api.lethalmon-fangame.com`** — a WebSocket connection for the live
  online player count.

Every downloaded launcher update is verified against an ed25519 signature
before being applied (see `app_launcher_update.go` and
`internal/updatekey`), and the update itself is installed by having the
newly downloaded build briefly relaunch itself to move its own binary into
place — no shell scripts, no `cmd.exe`, no third-party installer involved.

## Project structure

```
app*.go                    Backend, split by feature area:
  app.go                     App struct / Wails startup
  app_config.go               Launcher's own persisted settings
  app_download.go             Game download + zip extraction
  app_game.go                  Install detection, launch, process tracking
  app_gameopts.go               Game's .gameopts read/write
  app_gpu_*.go                   GPU detection / per-app GPU preference (Windows-only)
  app_install.go                  Move/uninstall the game install
  app_launcher_update*.go          Self-update check, download, verify, apply
  app_vcredist_*.go                 VC++ Redistributable auto-install (Windows-only)
  app_exec_*.go                      OS-specific process helpers
releases.go                 GitHub Releases API client (shared by game + launcher update checks)
internal/updatekey/         Public half of the ed25519 update-signing keypair
tools/updatesign/           Standalone CLI to generate the keypair and sign release artifacts
frontend/                   React/TypeScript UI (Vite), calls the Go backend via
                             the generated bindings in frontend/wailsjs/
```

Files suffixed `_windows.go` / `_other.go` are Go build-tag pairs: the
Windows-only implementation and its no-op stub for other platforms (the
launcher currently targets Windows, but the backend stays cross-platform
where it costs nothing to).

## Prerequisites

- [Go](https://go.dev/) 1.23+
- [Node.js](https://nodejs.org/) 20+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation) v2.11.0:
  `go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0`

## Development

```
git clone https://github.com/Otaku17/lethalmon-launcher-update.git
cd lethalmon-launcher-update
wails dev
```

No separate `npm install` step needed — Wails runs `frontend:install` (see
`wails.json`) for you before the first launch. `wails dev` then runs the
app with hot reload on the frontend (Vite dev server) and exposes the Go
backend on `http://localhost:34115`, so browser devtools can inspect the
UI and call bound Go methods directly from the console.

**Local state**, useful to know when testing install/reset flows:
- Launcher settings: `%AppData%\LethalmonLauncher\config.json`
- Default game install location: `%UserProfile%\LethalmonLauncher\Game`

Deleting the config file resets the launcher to a first-run state (no
custom install dir, no legacy-install detection override) without
touching an actual game install.

**Testing the self-update flow** without the production signing key: run
`go run ./tools/updatesign -gen` in a scratch directory, swap the
generated public key into `internal/updatekey/updatekey.go` (don't commit
this), sign a locally built exe with the matching private key, and serve
it from a throwaway GitHub release or fork. `UpdateLauncher` refuses
anything not signed by the key baked into the binary it's running from,
so this is the only way to exercise the full flow end-to-end without
access to the real production key.

## Building

```
wails build
```

Produces a production build in `build/bin/Lethal Launcher.exe` (per
`wails.json`'s `outputfilename`). Published releases rename this to
`lethalmon-launcher.exe` — see `release.yml` — so the download URL never
changes between versions; a local build keeps the spaced name.

## CI/CD

- **`.github/workflows/build.yml`** — on every push/PR to `main`: installs
  dependencies, builds the frontend, runs `go vet`, builds the app, and
  uploads the resulting `.exe` as a workflow artifact. This is the fastest
  way to confirm a change builds cleanly without installing anything
  locally.
- **`.github/workflows/release.yml`** — on pushing a version tag (`v1.2.3`
  or `1.2.3`): builds the launcher, signs it with the ed25519 update key
  (`tools/updatesign`), verifies the signature it just produced, and
  publishes the `.exe` and its `.sig` to a GitHub release. The tag must
  match the version in `frontend/package.json`, or the release fails
  before anything is published — see the workflow's comments for why.

## Contributing

Issues and pull requests are welcome. A few things that keep changes easy
to review:

- Keep platform-specific code behind the existing `_windows.go` /
  `_other.go` split rather than runtime `if runtime.GOOS == ...` checks.
- `go vet ./...` and a `wails build` should both pass before opening a PR
  (the `build.yml` workflow checks this automatically).
- If a change touches the self-update or install flow, explain in the PR
  description what it changes about the launcher's on-disk/network
  behavior — see [How it works](#how-it-works) for what's expected to stay
  true.

## License

Distributed under the MIT License. See [LICENSE](LICENSE) for details.
