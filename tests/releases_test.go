package tests

import (
	"testing"

	"LethalmonLauncher/backend"
)

func TestTrimVersion(t *testing.T) {
	cases := map[string]string{
		"v1.3.9":   "1.3.9",
		"1.3.9":    "1.3.9",
		"  v2.0  ": "2.0",
		"":         "",
		// Only a leading "v" is stripped, so a tag that legitimately starts
		// with another letter isn't silently mangled.
		"version1.0": "ersion1.0",
	}

	for in, want := range cases {
		if got := backend.TrimVersion(in); got != want {
			t.Errorf("backend.TrimVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.3.9", "1.3.9", 0},
		{"1.3.10", "1.3.9", 1},
		{"1.3.9", "1.3.10", -1},

		// The reason this doesn't compare strings: "1.10.0" sorts before
		// "1.9.0" lexically, which would hide an update from every player on
		// 1.9.x once the launcher hit 1.10.
		{"1.10.0", "1.9.0", 1},
		{"1.9.0", "1.10.0", -1},

		// Tags may or may not carry the "v", on either side.
		{"v1.4.0", "1.3.9", 1},
		{"1.4.0", "v1.4.0", 0},

		// Missing trailing segments count as zero, so "1.4" and "1.4.0" are
		// the same version rather than an endless update prompt.
		{"1.4", "1.4.0", 0},
		{"1.4.1", "1.4", 1},
	}

	for _, c := range cases {
		if got := backend.CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("backend.CompareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// TestCompareVersionsTreatsUnparseableAsOldest pins a sharp edge rather than
// endorsing it: a version string that isn't purely dotted-numeric parses to no
// segments and compares as 0.0.0.
//
// The consequence is worth stating plainly. Publishing the game or the
// launcher under a tag like "v1.4.1-hotfix" makes backend.CompareVersions(latest,
// current) negative, so GetGameVersionCheck reports UpdateAvailable: false and
// every player silently stops being offered the update — no error, nothing in
// the UI, just a launcher that quietly believes it's current.
//
// It cuts the other way too: a player whose version.txt got corrupted into
// something unparseable is treated as being on 0.0.0, so the next check offers
// them the update and the install repairs itself. That self-healing is why
// this isn't simply "return 0 when either side is unparseable".
func TestCompareVersionsTreatsUnparseableAsOldest(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"nightly", "1.4.0", -1},
		{"1.4.0", "nightly", 1},
		{"1.4.0-rc1", "1.4.0", -1},
		{"1.4.0", "1.4.0-rc1", 1},
		{"nightly", "unstable", 0},
	}

	for _, c := range cases {
		if got := backend.CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("backend.CompareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestParseVersionParts(t *testing.T) {
	got := backend.ParseVersionParts("v1.10.3")
	want := []int{1, 10, 3}

	if len(got) != len(want) {
		t.Fatalf("backend.ParseVersionParts(\"v1.10.3\") = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("backend.ParseVersionParts(\"v1.10.3\") = %v, want %v", got, want)
		}
	}

	for _, in := range []string{"", "1.4.0-rc1", "1..0", "1.x.0", "latest"} {
		if parts := backend.ParseVersionParts(in); parts != nil {
			t.Errorf("backend.ParseVersionParts(%q) = %v, want nil", in, parts)
		}
	}
}

func TestPickGameDownloadURL(t *testing.T) {
	assets := []backend.ReleaseAsset{
		{Name: "changelog.md", BrowserDownloadURL: "https://example.test/changelog.md"},
		{Name: "Lethalmon.zip", BrowserDownloadURL: "https://example.test/Lethalmon.zip"},
		{Name: "extra.zip", BrowserDownloadURL: "https://example.test/extra.zip"},
	}

	if got := backend.PickGameDownloadURL(assets); got != "https://example.test/Lethalmon.zip" {
		t.Errorf("backend.PickGameDownloadURL() = %q, want the first .zip", got)
	}

	if got := backend.PickGameDownloadURL(nil); got != "" {
		t.Errorf("backend.PickGameDownloadURL(nil) = %q, want empty", got)
	}

	noZip := []backend.ReleaseAsset{{Name: "notes.txt", BrowserDownloadURL: "https://example.test/notes.txt"}}
	if got := backend.PickGameDownloadURL(noZip); got != "" {
		t.Errorf("backend.PickGameDownloadURL(no zip) = %q, want empty", got)
	}
}
