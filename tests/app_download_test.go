package tests

import (
	"archive/zip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"LethalmonLauncher/backend"
)

// fastDownloadRetries removes the production backoff and idle timeout for the
// duration of a test, so the retry and stall paths can be exercised in
// milliseconds instead of minutes.
func fastDownloadRetries(t *testing.T, idle time.Duration) {
	t.Helper()

	origDelay, origIdle := backend.DownloadRetryDelay, backend.DownloadIdleTimeout
	backend.DownloadRetryDelay = func(int) time.Duration { return 0 }
	backend.DownloadIdleTimeout = idle

	t.Cleanup(func() {
		backend.DownloadRetryDelay, backend.DownloadIdleTimeout = origDelay, origIdle
	})
}

// TestSafeExtractPathRejectsEscapes is the zip-slip guard: a crafted archive
// must not be able to write outside the install directory.
func TestSafeExtractPathRejectsEscapes(t *testing.T) {
	dest := filepath.Join("C:", "Games", "Lethalmon")
	if os.PathSeparator == '/' {
		dest = "/games/lethalmon"
	}

	escapes := []string{
		"../evil.exe",
		"../../evil.exe",
		"data/../../evil.exe",
		"../",
		"../lethalmon-sibling/evil.exe",
		"a/b/c/../../../../evil.exe",
	}

	for _, name := range escapes {
		t.Run(name, func(t *testing.T) {
			got, err := backend.SafeExtractPath(dest, name)
			if err == nil {
				t.Fatalf("backend.SafeExtractPath(%q, %q) = %q, want an error", dest, name, got)
			}
			if !strings.Contains(err.Error(), "invalid file path in archive") {
				t.Errorf("error = %q, want it to name the offending archive entry", err)
			}
		})
	}
}

func TestSafeExtractPathAllowsNormalEntries(t *testing.T) {
	dest := filepath.Join("C:", "Games", "Lethalmon")
	if os.PathSeparator == '/' {
		dest = "/games/lethalmon"
	}

	cases := map[string]string{
		"version.txt":            filepath.Join(dest, "version.txt"),
		"Saves/slot1.dat":        filepath.Join(dest, "Saves", "slot1.dat"),
		"Data/Audio/BGM/x.ogg":   filepath.Join(dest, "Data", "Audio", "BGM", "x.ogg"),
		"./version.txt":          filepath.Join(dest, "version.txt"),
		"Data/../version.txt":    filepath.Join(dest, "version.txt"),
		"Data/./Audio/theme.ogg": filepath.Join(dest, "Data", "Audio", "theme.ogg"),
	}

	for name, want := range cases {
		got, err := backend.SafeExtractPath(dest, name)
		if err != nil {
			t.Errorf("backend.SafeExtractPath(%q, %q) errored: %v", dest, name, err)
			continue
		}
		if got != want {
			t.Errorf("backend.SafeExtractPath(%q, %q) = %q, want %q", dest, name, got, want)
		}
	}
}

// writeTestZip builds a zip on disk from name->content and returns its path.
func writeTestZip(t *testing.T, entries map[string]string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "archive.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	return path
}

// TestFileMatchesZipEntry covers what RepairGame leans on: a file is left
// alone only when it is byte-for-byte what the archive says it should be.
func TestFileMatchesZipEntry(t *testing.T) {
	const content = "the original game file"

	zipPath := writeTestZip(t, map[string]string{"data.dat": content})
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()

	entry := reader.File[0]
	dir := t.TempDir()

	cases := []struct {
		name  string
		setup func(path string)
		want  bool
	}{
		{
			name:  "identical file is kept",
			setup: func(p string) { mustWrite(t, p, content) },
			want:  true,
		},
		{
			// Same length, different bytes: only the checksum catches this,
			// which is the point of hashing rather than stat-ing.
			name:  "corrupted file of the same size is replaced",
			setup: func(p string) { mustWrite(t, p, "the 0riginal game file") },
			want:  false,
		},
		{
			name:  "truncated file is replaced",
			setup: func(p string) { mustWrite(t, p, content[:5]) },
			want:  false,
		},
		{
			name:  "missing file is replaced",
			setup: func(string) {},
			want:  false,
		},
		{
			name:  "directory in the way is replaced",
			setup: func(p string) { mustMkdir(t, p) },
			want:  false,
		},
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(dir, fmt.Sprintf("case%d.dat", i))
			c.setup(path)

			if got := backend.FileMatchesZipEntry(path, entry); got != c.want {
				t.Errorf("backend.FileMatchesZipEntry() = %v, want %v", got, c.want)
			}
		})
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

// bigBody is large enough that an interrupted transfer leaves a meaningful
// partial file to resume from.
var bigBody = strings.Repeat("lethalmon", 5000)

// TestDownloadFileResumesAfterInterruption is the behavior the retry logic
// exists for: a connection that drops partway through must pick up where it
// left off rather than restarting, so a flaky link can still finish a large
// download.
func TestDownloadFileResumesAfterInterruption(t *testing.T) {
	fastDownloadRetries(t, 5*time.Second)

	const firstChunk = 4000
	var ranges []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		ranges = append(ranges, rangeHeader)

		if rangeHeader == "" {
			// First attempt: hand over a prefix, then drop the connection.
			w.Header().Set("Content-Length", fmt.Sprint(len(bigBody)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(bigBody[:firstChunk]))
			w.(http.Flusher).Flush()
			panic(http.ErrAbortHandler)
		}

		var offset int
		if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &offset); err != nil {
			t.Errorf("malformed Range header %q", rangeHeader)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, len(bigBody)-1, len(bigBody)))
		w.Header().Set("Content-Length", fmt.Sprint(len(bigBody)-offset))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte(bigBody[offset:]))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "download.bin")
	if err := backend.DownloadFile(srv.URL, dest, func(int) {}); err != nil {
		t.Fatalf("backend.DownloadFile() error: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(got) != bigBody {
		t.Fatalf("downloaded %d bytes, want %d", len(got), len(bigBody))
	}

	if len(ranges) != 2 {
		t.Fatalf("server saw %d requests (%v), want 2", len(ranges), ranges)
	}
	if want := fmt.Sprintf("bytes=%d-", firstChunk); ranges[1] != want {
		t.Errorf("retry sent Range %q, want %q — it restarted instead of resuming", ranges[1], want)
	}
}

// TestDownloadFileRestartsWhenRangeIsIgnored covers the other half: a server
// that answers a Range request with the whole body anyway. The bytes already
// on disk are then meaningless, and appending to them would silently produce a
// corrupt archive.
func TestDownloadFileRestartsWhenRangeIsIgnored(t *testing.T) {
	fastDownloadRetries(t, 5*time.Second)

	const firstChunk = 4000
	attempts := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Content-Length", fmt.Sprint(len(bigBody)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(bigBody[:firstChunk]))
			w.(http.Flusher).Flush()
			panic(http.ErrAbortHandler)
		}

		// Range ignored: full body, status 200.
		w.Header().Set("Content-Length", fmt.Sprint(len(bigBody)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(bigBody))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "download.bin")
	if err := backend.DownloadFile(srv.URL, dest, func(int) {}); err != nil {
		t.Fatalf("backend.DownloadFile() error: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(got) != bigBody {
		t.Fatalf("downloaded %d bytes, want %d — the partial prefix was not discarded", len(got), len(bigBody))
	}
}

// TestDownloadFileDetectsStall covers the idle timeout: a server that accepts
// the connection and then goes quiet must be abandoned, not waited on forever.
func TestDownloadFileDetectsStall(t *testing.T) {
	fastDownloadRetries(t, 50*time.Millisecond)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(bigBody)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(bigBody[:10]))
		w.(http.Flusher).Flush()

		// Go silent until the client gives up and hangs up.
		<-r.Context().Done()
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "download.bin")
	err := backend.DownloadFile(srv.URL, dest, func(int) {})
	if err == nil {
		t.Fatal("backend.DownloadFile() returned nil for a connection that never finished")
	}
	if !strings.Contains(err.Error(), "stalled") {
		t.Errorf("error = %q, want it to report a stalled connection", err)
	}
}

func TestDownloadFileGivesUpAfterMaxAttempts(t *testing.T) {
	fastDownloadRetries(t, 5*time.Second)

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "download.bin")
	err := backend.DownloadFile(srv.URL, dest, func(int) {})
	if err == nil {
		t.Fatal("backend.DownloadFile() returned nil despite every attempt failing")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %q, want it to carry the HTTP status", err)
	}
	if requests != backend.DownloadMaxAttempts {
		t.Errorf("server saw %d requests, want %d", requests, backend.DownloadMaxAttempts)
	}
}

func TestDownloadFileReportsProgress(t *testing.T) {
	fastDownloadRetries(t, 5*time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(bigBody)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(bigBody))
	}))
	defer srv.Close()

	var percents []int
	dest := filepath.Join(t.TempDir(), "download.bin")
	if err := backend.DownloadFile(srv.URL, dest, func(p int) { percents = append(percents, p) }); err != nil {
		t.Fatalf("backend.DownloadFile() error: %v", err)
	}

	if len(percents) == 0 {
		t.Fatal("no progress was reported")
	}
	if last := percents[len(percents)-1]; last != 100 {
		t.Errorf("final progress = %d%%, want 100%%", last)
	}
	for i := 1; i < len(percents); i++ {
		if percents[i] < percents[i-1] {
			t.Fatalf("progress went backwards: %d%% then %d%%", percents[i-1], percents[i])
		}
	}
}
