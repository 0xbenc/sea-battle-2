// Package tunnel owns an optional, anonymous public connection for one game.
package tunnel

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const release = "2026.10.1"

type asset struct{ name, sha string }

// SHA-256 digests published by the official GitHub release API for this version.
var assets = map[string]asset{
	"linux/amd64":   {"cloudflared-linux-amd64", "3bfc539f45d2620c71debb7a86be0efa737ed5ccbad9eb85c787116900e5c586"},
	"linux/arm64":   {"cloudflared-linux-arm64", "2c65dfabf616c40efc7545d107d7a57ad5e21377e2964b7e88af19a238074a46"},
	"darwin/amd64":  {"cloudflared-darwin-amd64.tgz", "3b0dbaf378337e9d7fb6cd235a76d1d6a7dee434c2d8edf9281164fb6a3837ab"},
	"darwin/arm64":  {"cloudflared-darwin-arm64.tgz", "647615df99d49cd179ea5e13ef760b9bbfb33630f6ae62f87e33dfadda60adcc"},
	"windows/amd64": {"cloudflared-windows-amd64.exe", "4d2490e74082c5d48dbfedaf92576fa5bd1702379ab88eb7faa0cfbe8dc891b6"},
}

func helper(ctx context.Context, report func(string)) (string, error) {
	if path := os.Getenv("CLOUDFLARED"); path != "" {
		return exec.LookPath(path)
	}
	name := "cloudflared"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	exe, _ := os.Executable()
	for _, path := range []string{filepath.Join(filepath.Dir(exe), name), filepath.Join("bin", name)} {
		if p, err := exec.LookPath(path); err == nil {
			return filepath.Abs(p)
		}
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	a, ok := assets[runtime.GOOS+"/"+runtime.GOARCH]
	if !ok {
		return "", errors.New("automatic public connections are unavailable on this platform")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cache, "sea-battle-2", "tools", release)
	path := filepath.Join(dir, name)
	if p, err := exec.LookPath(path); err == nil {
		return p, nil
	}
	report("Preparing public connection support (first use)…")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return path, download(ctx, http.DefaultClient, "https://github.com/cloudflare/cloudflared/releases/download/"+release+"/"+a.name, a, path)
}

func download(ctx context.Context, client *http.Client, url string, a asset, destination string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("helper download: HTTP %d", resp.StatusCode)
	}
	archive, err := os.CreateTemp(filepath.Dir(destination), ".download-*")
	if err != nil {
		return err
	}
	defer os.Remove(archive.Name())
	defer archive.Close()
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(archive, hash), io.LimitReader(resp.Body, 80<<20+1))
	if err != nil {
		return err
	}
	if n > 80<<20 {
		return errors.New("helper download exceeds size limit")
	}
	if hex.EncodeToString(hash.Sum(nil)) != a.sha {
		return errors.New("helper download checksum mismatch")
	}
	if _, err = archive.Seek(0, io.SeekStart); err != nil {
		return err
	}
	var source io.Reader = archive
	if strings.HasSuffix(a.name, ".tgz") {
		gz, err := gzip.NewReader(archive)
		if err != nil {
			return err
		}
		defer gz.Close()
		tr := tar.NewReader(gz)
		for {
			h, e := tr.Next()
			if e != nil {
				return errors.New("helper executable missing from archive")
			}
			if h.Name == "cloudflared" && h.Typeflag == tar.TypeReg {
				source = tr
				break
			}
		}
	}
	binary, err := os.CreateTemp(filepath.Dir(destination), ".helper-*")
	if err != nil {
		return err
	}
	defer os.Remove(binary.Name())
	defer binary.Close()
	n, err = io.Copy(binary, io.LimitReader(source, 100<<20+1))
	if err != nil {
		return err
	}
	if n == 0 || n > 100<<20 {
		return errors.New("invalid helper executable size")
	}
	if err = binary.Chmod(0755); err != nil {
		return err
	}
	if err = binary.Close(); err != nil {
		return err
	}
	return os.Rename(binary.Name(), destination)
}
