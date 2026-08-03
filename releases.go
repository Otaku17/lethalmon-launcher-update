package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const gameReleasesRepo = "bfloo/lethalmon-launcher"

// launcherReleasesRepo only supplies the launcher's changelog/patch notes —
// the actual update file is served by the distribution server instead (see
// app_launcher_update.go), not GitHub releases.
const launcherReleasesRepo = "Otaku17/lethalmon-launcher-update"

// ReleaseAsset is a single downloadable file attached to a GitHub release.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Release represents a single GitHub release entry shown in the changelog.
type Release struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	PublishedAt time.Time      `json:"published_at"`
	HTMLURL     string         `json:"html_url"`
	Assets      []ReleaseAsset `json:"assets"`
}

// GameVersionCheck reports whether a newer game version is available on the
// game's GitHub releases page.
type GameVersionCheck struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	DownloadURL     string `json:"downloadUrl,omitempty"`
	ReleaseURL      string `json:"releaseUrl,omitempty"`
}

// fetchReleases lists every GitHub release for repo, newest first. An empty
// repo (e.g. launcherReleasesRepo before it's configured) returns an empty
// list rather than an error.
func fetchReleases(repo string) ([]Release, error) {
	if repo == "" {
		return []Release{}, nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github api returned %d: %s", resp.StatusCode, body)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	return releases, nil
}

// GetGameReleases fetches the list of releases from the game's GitHub repository.
func (a *App) GetGameReleases() ([]Release, error) {
	return fetchReleases(gameReleasesRepo)
}

// GetLauncherReleases fetches the list of releases from the launcher's
// GitHub repository.
func (a *App) GetLauncherReleases() ([]Release, error) {
	return fetchReleases(launcherReleasesRepo)
}

// GetGameVersionCheck compares the installed game version (from
// version.txt) against the game's latest GitHub release, so the launcher
// can prompt the user to update.
func (a *App) GetGameVersionCheck() (GameVersionCheck, error) {
	current, err := a.GetGameVersion()
	if err != nil {
		return GameVersionCheck{}, err
	}

	releases, err := fetchReleases(gameReleasesRepo)
	if err != nil {
		return GameVersionCheck{}, err
	}
	if len(releases) == 0 {
		return GameVersionCheck{CurrentVersion: current}, nil
	}

	latestRelease := releases[0]
	latest := trimVersion(latestRelease.TagName)

	var downloadURL string
	for _, asset := range latestRelease.Assets {
		if strings.HasSuffix(asset.Name, ".zip") {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	return GameVersionCheck{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: current == "" || compareVersions(latest, current) > 0,
		DownloadURL:     downloadURL,
		ReleaseURL:      latestRelease.HTMLURL,
	}, nil
}

// trimVersion strips a leading "v" from a git tag, e.g. "v1.3.9" -> "1.3.9".
func trimVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// compareVersions returns 1 if a > b, -1 if a < b, 0 if equal (or not
// comparable). Dotted numeric versions only (e.g. "1.10.0" > "1.9.0").
func compareVersions(a, b string) int {
	partsA := parseVersionParts(a)
	partsB := parseVersionParts(b)

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		var va, vb int
		if i < len(partsA) {
			va = partsA[i]
		}
		if i < len(partsB) {
			vb = partsB[i]
		}

		switch {
		case va > vb:
			return 1
		case va < vb:
			return -1
		}
	}

	return 0
}

// parseVersionParts splits a dotted version string into numeric segments
// (e.g. "1.10.0" -> [1, 10, 0]), returning nil if any segment isn't a
// plain integer.
func parseVersionParts(v string) []int {
	v = trimVersion(v)
	if v == "" {
		return nil
	}

	raw := strings.Split(v, ".")
	parts := make([]int, 0, len(raw))

	for _, p := range raw {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		parts = append(parts, n)
	}

	return parts
}
