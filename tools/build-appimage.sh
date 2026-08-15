#!/usr/bin/env bash
# Packages the Linux build already produced by
# `wails build -platform linux/amd64 -nopackage` (see .github/workflows/
# build.yml and release.yml) into a self-contained AppImage: GTK3 and
# WebKit2GTK are bundled in via linuxdeploy + linuxdeploy-plugin-gtk, so the
# result runs on any distro without those needing to be preinstalled —
# unlike Wine, which the game still needs the player to install themselves
# (see the "wine required" flow in HomePage.tsx): that's a per-game runtime
# choice, not this launcher's own GTK/WebKit dependency.
#
# Shared between build.yml (every push/PR, as a plain workflow artifact) and
# release.yml (tagged releases, as a downloadable asset) so the exact same
# packaging path is exercised well before a real release depends on it.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

bin_path="build/bin/Lethal Launcher"
if [ ! -f "$bin_path" ]; then
	echo "error: $bin_path not found — run 'wails build -platform linux/amd64 -nopackage' first" >&2
	exit 1
fi

tools_dir="$repo_root/.appimage-tools"
mkdir -p "$tools_dir"

fetch_tool() {
	local name="$1" url="$2" dest="$tools_dir/$1"
	if [ ! -x "$dest" ]; then
		echo "Downloading $name..."
		wget -q -O "$dest" "$url"
		chmod +x "$dest"
	fi
}

fetch_tool linuxdeploy \
	"https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-x86_64.AppImage"
# Unlike linuxdeploy itself, this plugin isn't published as a compiled
# AppImage release asset — it's a plain bash script fetched straight from
# the repo (see its README), and linuxdeploy's plugin lookup expects it
# under this exact "linuxdeploy-plugin-<name>.sh" name on PATH.
fetch_tool linuxdeploy-plugin-gtk.sh \
	"https://raw.githubusercontent.com/linuxdeploy/linuxdeploy-plugin-gtk/master/linuxdeploy-plugin-gtk.sh"
fetch_tool appimagetool \
	"https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage"

export PATH="$tools_dir:$PATH"
# GitHub Actions runners don't reliably support FUSE-mounting AppImages
# (sandboxed user namespaces), so extract-and-run instead of mounting.
export APPIMAGE_EXTRACT_AND_RUN=1

appdir="$repo_root/build/bin/AppDir"
rm -rf "$appdir"
mkdir -p "$appdir/usr/bin"
cp "$bin_path" "$appdir/usr/bin/lethalmon-launcher"
cp build/appicon.png "$appdir/lethalmon-launcher.png"
cp build/linux/lethalmon-launcher.desktop "$appdir/lethalmon-launcher.desktop"

work_dir="$(mktemp -d)"
(
	cd "$work_dir"
	DEPLOY_GTK_VERSION=3 linuxdeploy \
		--appdir "$appdir" \
		--executable "$appdir/usr/bin/lethalmon-launcher" \
		--desktop-file "$appdir/lethalmon-launcher.desktop" \
		--icon-file "$appdir/lethalmon-launcher.png" \
		--plugin gtk \
		--output appimage
)

# linuxdeploy names the AppImage after the .desktop file's Name + arch
# (exact scheme isn't a stable contract), so pick up whatever it produced
# instead of hardcoding a guess, and settle on one predictable name for
# callers (build.yml's artifact glob, release.yml's upload step).
produced=("$work_dir"/*.AppImage)
if [ ! -e "${produced[0]}" ]; then
	echo "error: linuxdeploy did not produce an .AppImage in $work_dir" >&2
	exit 1
fi
mv "${produced[0]}" "$repo_root/build/bin/lethalmon-launcher-x86_64.AppImage"
rm -rf "$appdir" "$work_dir"
