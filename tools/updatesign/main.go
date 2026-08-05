// Command updatesign manages the ed25519 keypair used to sign launcher
// update artifacts, and signs them before upload to the distribution
// server. This is a standalone build-time tool — it is never bundled into
// the launcher itself (see app_launcher_update.go for the verification
// side, which only needs the public key from internal/updatekey).
//
// Usage:
//
//	go run ./tools/updatesign -gen
//	go run ./tools/updatesign -sign "build/bin/Lethal Launcher.exe" -key launcher_update_private.key
//	go run ./tools/updatesign -verify "build/bin/Lethal Launcher.exe"
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strings"

	"LethalmonLauncher/internal/updatekey"
)

// main dispatches to -gen or -sign based on the flags given; it exits with
// a usage message if neither is provided.
func main() {
	gen := flag.Bool("gen", false, "generate a new ed25519 keypair")
	sign := flag.String("sign", "", "path to the file to sign")
	verify := flag.String("verify", "", "path to a file to check against its \".sig\" companion")
	keyPath := flag.String("key", "launcher_update_private.key", "path to the private key file (base64), used with -sign")
	flag.Parse()

	switch {
	case *gen:
		generateKeypair()
	case *sign != "":
		signFile(*sign, *keyPath)
	case *verify != "":
		verifyFile(*verify)
	default:
		fmt.Fprintln(os.Stderr, "usage: updatesign -gen | updatesign -sign <file> [-key <privatekey.b64>] | updatesign -verify <file>")
		os.Exit(1)
	}
}

// generateKeypair creates a new ed25519 keypair and writes both halves to
// disk as base64 text files in the current directory.
func generateKeypair() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fatal("generate key", err)
	}

	privPath := "launcher_update_private.key"
	pubPath := "launcher_update_public.key"

	// Silently overwriting an existing keypair would destroy the only key
	// installed launchers accept updates from. Since verification is
	// mandatory, that leaves every existing install unable to auto-update
	// ever again — recoverable only by asking players to reinstall by
	// hand. Never worth doing by accident.
	var existing []string
	for _, path := range []string{privPath, pubPath} {
		if _, err := os.Stat(path); err == nil {
			existing = append(existing, path)
		}
	}
	if len(existing) > 0 {
		fmt.Fprintf(os.Stderr, "refusing to overwrite: %s\n", strings.Join(existing, ", "))
		fmt.Fprintln(os.Stderr, "Replacing the update keypair means no already-installed launcher can verify future updates.")
		fmt.Fprintln(os.Stderr, "Move the existing files aside first if you really mean to generate a new keypair.")
		os.Exit(1)
	}

	if err := os.WriteFile(privPath, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
		fatal("write private key", err)
	}
	if err := os.WriteFile(pubPath, []byte(base64.StdEncoding.EncodeToString(pub)), 0o644); err != nil {
		fatal("write public key", err)
	}

	fmt.Printf("Generated %s (KEEP SECRET — do not commit, back it up somewhere safe) and %s.\n", privPath, pubPath)
	fmt.Println("Paste the public key's content into PublicKeyB64 in internal/updatekey/updatekey.go.")
	fmt.Println("Until that constant matches, -sign will refuse to sign with this key.")
}

// signFile signs the file at path with the base64-encoded private key found
// at keyPath, writing the result as "<path>.sig".
func signFile(path, keyPath string) {
	keyB64, err := os.ReadFile(keyPath)
	if err != nil {
		fatal("read private key", err)
	}

	// The key file is base64 text and routinely carries a trailing newline
	// — an editor adds one, and so does a CI secret written out to disk by
	// the release workflow. Decoding without trimming would reject a
	// perfectly valid key.
	priv, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(keyB64)))
	if err != nil || len(priv) != ed25519.PrivateKeySize {
		fatal("invalid private key", err)
	}

	if err := checkKeyMatchesLauncher(ed25519.PrivateKey(priv)); err != nil {
		fatal("private key check", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fatal("read file to sign", err)
	}

	signature := ed25519.Sign(ed25519.PrivateKey(priv), data)

	sigPath := path + ".sig"
	if err := os.WriteFile(sigPath, []byte(base64.StdEncoding.EncodeToString(signature)), 0o644); err != nil {
		fatal("write signature", err)
	}

	fmt.Println("Wrote", sigPath, "— upload it alongside", path, "to the distribution server.")
}

// verifyFile checks the file at path against its "<path>.sig" companion
// using the public key shipped in launcher builds — the exact check an
// installed launcher runs before applying an update.
//
// Running it at the end of the release workflow catches anything that
// changed the exe's bytes after it was signed, which silently invalidates
// the signature. Authenticode signing is the one to watch for: signtool
// rewrites the executable, so it has to run BEFORE updatesign, never
// after.
func verifyFile(path string) {
	pub, err := updatekey.Public()
	if err != nil {
		fatal("public key", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fatal("read file to verify", err)
	}

	sigPath := path + ".sig"
	sigB64, err := os.ReadFile(sigPath)
	if err != nil {
		fatal("read signature", err)
	}

	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigB64)))
	if err != nil {
		fatal("invalid signature encoding", err)
	}

	if !ed25519.Verify(pub, data, signature) {
		fmt.Fprintf(os.Stderr, "%s does not sign %s — an installed launcher would refuse this update\n", sigPath, path)
		os.Exit(1)
	}

	fmt.Println("OK:", sigPath, "signs", path, "and matches the public key shipped in the launcher.")
}

// checkKeyMatchesLauncher reports whether priv is the private half of the
// public key shipped inside launcher builds (see internal/updatekey).
//
// Verification is mandatory on the launcher side, so an update signed with
// the wrong private key is one that every installed launcher refuses to
// install — and the only way back from that is asking players to
// reinstall by hand. Signing the wrong thing is a mistake worth catching
// here, before the artifact is ever published.
func checkKeyMatchesLauncher(priv ed25519.PrivateKey) error {
	expected, err := updatekey.Public()
	if err != nil {
		return err
	}

	actual, ok := priv.Public().(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("unexpected public key type %T", priv.Public())
	}

	if !actual.Equal(expected) {
		return fmt.Errorf(
			"this private key does not match the public key shipped in the launcher.\n"+
				"  launcher expects: %s\n"+
				"  this key derives: %s\n"+
				"Signing with it would produce an update every installed launcher rejects",
			updatekey.PublicKeyB64,
			base64.StdEncoding.EncodeToString(actual),
		)
	}

	return nil
}

// fatal prints an error prefixed with context to stderr and exits with a
// non-zero status.
func fatal(context string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", context, err)
	os.Exit(1)
}
