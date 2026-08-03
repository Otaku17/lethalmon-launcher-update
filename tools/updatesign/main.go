// Command updatesign manages the ed25519 keypair used to sign launcher
// update artifacts, and signs them before upload to the distribution
// server. This is a standalone build-time tool — it is never bundled into
// the launcher itself (see app_launcher_update.go for the verification
// side, which only needs the public key).
//
// Usage:
//
//	go run ./tools/updatesign -gen
//	go run ./tools/updatesign -sign "build/bin/Lethal Launcher.exe" -key launcher_update_private.key
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
)

// main dispatches to -gen or -sign based on the flags given; it exits with
// a usage message if neither is provided.
func main() {
	gen := flag.Bool("gen", false, "generate a new ed25519 keypair")
	sign := flag.String("sign", "", "path to the file to sign")
	keyPath := flag.String("key", "launcher_update_private.key", "path to the private key file (base64), used with -sign")
	flag.Parse()

	switch {
	case *gen:
		generateKeypair()
	case *sign != "":
		signFile(*sign, *keyPath)
	default:
		fmt.Fprintln(os.Stderr, "usage: updatesign -gen | updatesign -sign <file> [-key <privatekey.b64>]")
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

	if err := os.WriteFile(privPath, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
		fatal("write private key", err)
	}
	if err := os.WriteFile(pubPath, []byte(base64.StdEncoding.EncodeToString(pub)), 0o644); err != nil {
		fatal("write public key", err)
	}

	fmt.Printf("Generated %s (KEEP SECRET — do not commit, back it up somewhere safe) and %s.\n", privPath, pubPath)
	fmt.Println("Paste the public key's content into launcherUpdatePublicKeyB64 in app_launcher_update.go.")
}

// signFile signs the file at path with the base64-encoded private key found
// at keyPath, writing the result as "<path>.sig".
func signFile(path, keyPath string) {
	keyB64, err := os.ReadFile(keyPath)
	if err != nil {
		fatal("read private key", err)
	}

	priv, err := base64.StdEncoding.DecodeString(string(keyB64))
	if err != nil || len(priv) != ed25519.PrivateKeySize {
		fatal("invalid private key", err)
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

// fatal prints an error prefixed with context to stderr and exits with a
// non-zero status.
func fatal(context string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", context, err)
	os.Exit(1)
}
