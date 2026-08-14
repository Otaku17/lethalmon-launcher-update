package tests

import (
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"LethalmonLauncher/backend"
)

// signedFixture returns a keypair, a payload, and a valid base64 signature
// over it — the shape of what a release publishes (see tools/updatesign).
func signedFixture(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, []byte, []byte) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	payload := []byte("pretend this is lethalmon-launcher.exe")
	sig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload)))

	return pub, priv, payload, sig
}

func TestVerifyUpdateSignatureAcceptsGenuineSignature(t *testing.T) {
	pub, _, payload, sig := signedFixture(t)

	if err := backend.VerifyUpdateSignature(pub, payload, sig); err != nil {
		t.Fatalf("backend.VerifyUpdateSignature() rejected a genuine signature: %v", err)
	}
}

// TestVerifyUpdateSignatureToleratesWhitespace pins the one leniency in the
// check: a trailing newline is a formatting artifact, not evidence of
// tampering, and must not be reported as a failed signature.
func TestVerifyUpdateSignatureToleratesWhitespace(t *testing.T) {
	pub, _, payload, sig := signedFixture(t)

	for _, wrapped := range [][]byte{
		[]byte(string(sig) + "\n"),
		[]byte(string(sig) + "\r\n"),
		[]byte("  " + string(sig) + "  \n"),
	} {
		if err := backend.VerifyUpdateSignature(pub, payload, wrapped); err != nil {
			t.Errorf("backend.VerifyUpdateSignature() rejected %q: %v", wrapped, err)
		}
	}
}

// TestVerifyUpdateSignatureRejects is the test this whole file exists for.
// Every case below is an executable that must never reach
// installLauncherUpdate, because installing it would replace the running
// launcher on a player's machine.
func TestVerifyUpdateSignatureRejects(t *testing.T) {
	pub, _, payload, sig := signedFixture(t)
	otherPub, otherPriv, _, _ := signedFixture(t)
	_ = otherPub

	tampered := append([]byte{}, payload...)
	tampered[0] ^= 0xFF

	rawSig, err := base64.StdEncoding.DecodeString(string(sig))
	if err != nil {
		t.Fatalf("decode fixture signature: %v", err)
	}
	truncated := []byte(base64.StdEncoding.EncodeToString(rawSig[:len(rawSig)-1]))

	flipped := append([]byte{}, rawSig...)
	flipped[0] ^= 0x01
	flippedB64 := []byte(base64.StdEncoding.EncodeToString(flipped))

	wrongKeySig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(otherPriv, payload)))

	cases := []struct {
		name    string
		key     ed25519.PublicKey
		data    []byte
		sig     []byte
		wantErr string
	}{
		{"payload modified after signing", pub, tampered, sig, "verification failed"},
		{"signed by a different key", pub, payload, wrongKeySig, "verification failed"},
		{"signature bit flipped", pub, payload, flippedB64, "verification failed"},
		{"signature truncated", pub, payload, truncated, "verification failed"},
		{"signature empty", pub, payload, []byte(""), "verification failed"},
		{"signature not base64", pub, payload, []byte("this is not base64!!"), "encoding"},
		{"signature is an error page", pub, payload, []byte("<html>404 not found</html>"), "encoding"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := backend.VerifyUpdateSignature(c.key, c.data, c.sig)
			if err == nil {
				t.Fatal("backend.VerifyUpdateSignature() accepted an update it must refuse")
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("error = %q, want it to mention %q", err, c.wantErr)
			}
		})
	}
}

func TestFetchUpdateSignature(t *testing.T) {
	t.Run("serves the body verbatim", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("c2lnbmF0dXJl\n"))
		}))
		defer srv.Close()

		got, err := backend.FetchUpdateSignature(srv.URL)
		if err != nil {
			t.Fatalf("backend.FetchUpdateSignature() error: %v", err)
		}
		if string(got) != "c2lnbmF0dXJl\n" {
			t.Errorf("backend.FetchUpdateSignature() = %q", got)
		}
	})

	// A 404 must surface as an error rather than as empty signature bytes: an
	// empty signature that reached backend.VerifyUpdateSignature would be reported as
	// "verification failed", blaming the artifact for a missing file.
	t.Run("rejects a non-200 response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		if _, err := backend.FetchUpdateSignature(srv.URL); err == nil {
			t.Fatal("backend.FetchUpdateSignature() accepted a 404")
		}
	})

	t.Run("reports an unreachable host", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		url := srv.URL
		srv.Close()

		if _, err := backend.FetchUpdateSignature(url); err == nil {
			t.Fatal("backend.FetchUpdateSignature() accepted an unreachable host")
		}
	})
}

// TestVerifyLauncherUpdateSignatureUsesEmbeddedKey checks the wiring end to
// end against the key actually baked into the binary (internal/updatekey):
// whatever a release serves, it is refused unless it was signed with the
// matching private half.
func TestVerifyLauncherUpdateSignatureUsesEmbeddedKey(t *testing.T) {
	_, priv, payload, _ := signedFixture(t)
	forged := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(forged))
	}))
	defer srv.Close()

	if err := backend.VerifyLauncherUpdateSignature(srv.URL, payload); err == nil {
		t.Fatal("backend.VerifyLauncherUpdateSignature() accepted an update signed with a foreign key")
	}
}

func TestPickLauncherAssets(t *testing.T) {
	cases := []struct {
		name       string
		assets     []backend.ReleaseAsset
		wantDL     string
		wantSigURL string
	}{
		{
			name: "a complete release",
			assets: []backend.ReleaseAsset{
				{Name: "lethalmon-launcher.exe", BrowserDownloadURL: "https://example.test/l.exe"},
				{Name: "lethalmon-launcher.exe.sig", BrowserDownloadURL: "https://example.test/l.exe.sig"},
			},
			wantDL:     "https://example.test/l.exe",
			wantSigURL: "https://example.test/l.exe.sig",
		},
		{
			// Asset order is GitHub's to decide, so the signature appearing
			// first must not change the outcome.
			name: "signature listed first",
			assets: []backend.ReleaseAsset{
				{Name: "lethalmon-launcher.exe.sig", BrowserDownloadURL: "https://example.test/l.exe.sig"},
				{Name: "lethalmon-launcher.exe", BrowserDownloadURL: "https://example.test/l.exe"},
			},
			wantDL:     "https://example.test/l.exe",
			wantSigURL: "https://example.test/l.exe.sig",
		},
		{
			// The ".sig" must never be mistaken for the executable: that
			// would have UpdateLauncher verify an artifact against itself.
			name: "signature only",
			assets: []backend.ReleaseAsset{
				{Name: "lethalmon-launcher.exe.sig", BrowserDownloadURL: "https://example.test/l.exe.sig"},
			},
			wantDL:     "",
			wantSigURL: "https://example.test/l.exe.sig",
		},
		{
			// An unsigned release: UpdateLauncher refuses this outright.
			name: "executable only",
			assets: []backend.ReleaseAsset{
				{Name: "lethalmon-launcher.exe", BrowserDownloadURL: "https://example.test/l.exe"},
			},
			wantDL:     "https://example.test/l.exe",
			wantSigURL: "",
		},
		{
			name: "unrelated assets are ignored",
			assets: []backend.ReleaseAsset{
				{Name: "Lethalmon.zip", BrowserDownloadURL: "https://example.test/game.zip"},
				{Name: "notes.md", BrowserDownloadURL: "https://example.test/notes.md"},
			},
			wantDL:     "",
			wantSigURL: "",
		},
		{
			name:       "no assets at all",
			assets:     nil,
			wantDL:     "",
			wantSigURL: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotDL, gotSig := backend.PickLauncherAssets(c.assets)
			if gotDL != c.wantDL {
				t.Errorf("downloadURL = %q, want %q", gotDL, c.wantDL)
			}
			if gotSig != c.wantSigURL {
				t.Errorf("signatureURL = %q, want %q", gotSig, c.wantSigURL)
			}
		})
	}
}
