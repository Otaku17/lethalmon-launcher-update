// Package updatekey holds the ed25519 public key that launcher update
// artifacts are signed against.
//
// It is deliberately shared by both sides of the signing scheme: the
// launcher verifies a downloaded update against it (see
// app_launcher_update.go), and tools/updatesign checks — right after
// signing — that the private key it was handed is the matching half. That
// second use is what makes it worth a package of its own: verification is
// mandatory on the launcher side, so signing a release with the wrong
// private key would produce a release that every installed launcher
// refuses. Catching that at signing time turns a bricked auto-update into
// a failed build.
package updatekey

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

// PublicKeyB64 is the public half of the ed25519 keypair used to sign
// launcher update artifacts. It only lets a binary VERIFY that an update
// was signed by whoever holds the matching private key — it can't be used
// to sign anything, so embedding it in the shipped executable is safe.
//
// This protects against a compromised GitHub account or a tampered release
// asset serving a modified executable; it's independent of (and doesn't
// replace) Authenticode code signing, which is about OS-level trust
// (SmartScreen/Defender), not update integrity.
const PublicKeyB64 = "E1rqwTfPYmMI0R0bMMR3NuMegdfDkUpoWxO6xZsMSS0="

// Public decodes PublicKeyB64 into a usable ed25519 key, validating its
// length.
func Public() (ed25519.PublicKey, error) {
	key, err := base64.StdEncoding.DecodeString(PublicKeyB64)
	if err != nil {
		return nil, fmt.Errorf("invalid embedded public key: %w", err)
	}
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid embedded public key size: %d", len(key))
	}
	return ed25519.PublicKey(key), nil
}
