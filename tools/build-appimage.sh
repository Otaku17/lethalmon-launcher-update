#!/usr/bin/env bash
# Packages the Linux build already produced by
# `wails build -platform linux/amd64 -nopackage` (see .github/workflows/
# build.yml and release.yml) into an AppImage that links against the
# system's own GTK3/WebKit2GTK-4.1 stack instead of bundling it.
#
# An earlier version of this script bundled that whole stack in via
# linuxdeploy-plugin-gtk, on the theory that it would let the AppImage run
# on any distro without those libraries preinstalled. In practice it did the
# opposite: the bundled GTK/WebKit/Pango/Cairo build gets mixed at runtime
# with pieces of the *system's* GTK theme/rendering stack (GTK talks to the
# X/Wayland session and the system's widget theme regardless of which
# libgtk-3.so.0 loaded it), and that version mismatch is what produced a
# blank grey window on real installs — confirmed by the fact that forcing
# `LD_LIBRARY_PATH=/usr/lib` (system libs only, none of the bundled ones)
# fixed it. So this now excludes the entire GTK/GLib/WebKit dependency graph
# from bundling: the AppImage requires GTK3 and WebKit2GTK-4.1 to be
# installed on the host (e.g. `sudo pacman -S gtk3 webkit2gtk-4.1` on Arch,
# or the -dev packages this same script's CI callers already install), the
# same way it already requires Wine for the game itself (see the "wine
# required" flow in HomePage.tsx) — a per-system runtime dependency, not
# something bundled.
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

# linuxdeploy bundles every shared library the executable links against by
# default, which would pull GTK3/WebKit2GTK-4.1 and their entire dependency
# graph straight back in even without --plugin gtk. Each pattern below keeps
# one whole graph out — GTK/GLib's own stack, WebKit's, and the widget/text
# rendering libraries both depend on (Pango, Cairo, ATK, GdkPixbuf, HarfBuzz)
# — so nothing from it is bundled and nothing from it can end up mismatched
# against the system copies loaded alongside it at runtime (see the header
# comment for why a partial exclude reproduces the same bug this fixes).
exclude_libs=(
	'libgtk-3*' 'libgdk-3*' 'libgdk_pixbuf*'
	'libglib-2.0*' 'libgobject-2.0*' 'libgio-2.0*' 'libgmodule-2.0*' 'libgthread-2.0*'
	'libpango*' 'libcairo*' 'libatk*' 'libatspi*' 'libepoxy*'
	'libharfbuzz*' 'libfribidi*'
	'libwebkit2gtk-4.1*' 'libjavascriptcoregtk-4.1*' 'libsoup-3.0*' 'libwoff2*'
	'libnotify*' 'libsecret-1*'
	'libgstreamer*' 'libgst*'
)
exclude_args=()
for pattern in "${exclude_libs[@]}"; do
	exclude_args+=(--exclude-library "$pattern")
done

work_dir="$(mktemp -d)"
(
	cd "$work_dir"
	linuxdeploy \
		--appdir "$appdir" \
		--executable "$appdir/usr/bin/lethalmon-launcher" \
		--desktop-file "$appdir/lethalmon-launcher.desktop" \
		--icon-file "$appdir/lethalmon-launcher.png" \
		"${exclude_args[@]}" \
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
