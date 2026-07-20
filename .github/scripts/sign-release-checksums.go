// Command sign-release-checksums walks the release artifact directory, writes
// SHA256SUMS covering every regular file, then signs that file with the
// Ed25519 private key in $ATSTARTER_UPDATE_SIGNING_PRIVATE_KEY (base64) and
// drops SHA256SUMS.sig next to it. Both files are then attached to the
// GitHub Release so clients can verify updates before installing.
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: sign-release-checksums <artifact-dir>")
		os.Exit(2)
	}
	root := os.Args[1]

	privB64 := strings.TrimSpace(os.Getenv("ATSTARTER_UPDATE_SIGNING_PRIVATE_KEY"))
	if privB64 == "" {
		fmt.Fprintln(os.Stderr, "ATSTARTER_UPDATE_SIGNING_PRIVATE_KEY not set")
		os.Exit(1)
	}
	priv, err := base64.StdEncoding.DecodeString(privB64)
	if err != nil {
		fmt.Fprintln(os.Stderr, "decode private key:", err)
		os.Exit(1)
	}
	if len(priv) != ed25519.PrivateKeySize {
		fmt.Fprintf(os.Stderr, "private key size = %d, want %d\n", len(priv), ed25519.PrivateKeySize)
		os.Exit(1)
	}

	files, err := gatherFiles(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "walk:", err)
		os.Exit(1)
	}
	sort.Strings(files)

	var sb strings.Builder
	for _, rel := range files {
		abs := filepath.Join(root, rel)
		sum, err := sha256sum(abs)
		if err != nil {
			fmt.Fprintln(os.Stderr, "hash", rel, err)
			os.Exit(1)
		}
		// Match the sha256sum(1) format: <hex>␠␠<name>\n
		fmt.Fprintf(&sb, "%s  %s\n", sum, rel)
	}
	sums := []byte(sb.String())
	sumsPath := filepath.Join(root, "SHA256SUMS")
	if err := os.WriteFile(sumsPath, sums, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write SHA256SUMS:", err)
		os.Exit(1)
	}

	sig := ed25519.Sign(priv, sums)
	sigPath := filepath.Join(root, "SHA256SUMS.sig")
	if err := os.WriteFile(sigPath, []byte(base64.StdEncoding.EncodeToString(sig)+"\n"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write SHA256SUMS.sig:", err)
		os.Exit(1)
	}

	fmt.Println("signed", len(files), "files ->", sumsPath, "+", sigPath)
}

func gatherFiles(root string) ([]string, error) {
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// Skip anything already produced by an earlier sign pass.
		if rel == "SHA256SUMS" || rel == "SHA256SUMS.sig" {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	return out, err
}

func sha256sum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
