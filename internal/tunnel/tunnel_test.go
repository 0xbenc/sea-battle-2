package tunnel

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestVerifiedDownloadAndArchiveExtraction(t *testing.T) {
	for _, compressed := range []bool{false, true} {
		data := []byte("test executable")
		name := "binary"
		if compressed {
			var b bytes.Buffer
			gz := gzip.NewWriter(&b)
			tw := tar.NewWriter(gz)
			tw.WriteHeader(&tar.Header{Name: "cloudflared", Mode: 0755, Size: int64(len(data))})
			tw.Write(data)
			tw.Close()
			gz.Close()
			data = b.Bytes()
			name = "binary.tgz"
		}
		sum := sha256.Sum256(data)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
		target := filepath.Join(t.TempDir(), "cloudflared")
		a := asset{name, hex.EncodeToString(sum[:])}
		if err := download(context.Background(), server.Client(), server.URL, a, target); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(target)
		if err != nil || string(got) != "test executable" {
			t.Fatal(string(got), err)
		}
		a.sha = "incorrect"
		if download(context.Background(), server.Client(), server.URL, a, target) == nil {
			t.Fatal("accepted checksum mismatch")
		}
		got, _ = os.ReadFile(target)
		if string(got) != "test executable" {
			t.Fatal("failed download replaced working helper")
		}
		server.Close()
	}
}

func TestConnectorLifetimeAndCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	script := filepath.Join(t.TempDir(), "connector")
	// Shell loop uses no child processes, making ownership assertions unambiguous.
	body := "#!/bin/sh\necho 'https://test-fixture.trycloudflare.com'\necho 'Registered tunnel connection'\ntrap 'exit 0' INT TERM\nwhile :; do read line; done\n"
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
	h, err := launch(context.Background(), script, 1234, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if h.Address() != "wss://test-fixture.trycloudflare.com/sb2" {
		t.Fatal(h.Address())
	}
	h.Close()
	h.Close()
	select {
	case <-h.Done():
	default:
		t.Fatal("connector still running")
	}
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntrap 'exit 0' INT TERM\nwhile :; do read line; done\n"), 0755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if h, err := launch(ctx, script, 1234, time.Second); err == nil {
		h.Close()
		t.Fatal("started cancelled connector")
	}
	if h, err := launch(context.Background(), script, 1234, 20*time.Millisecond); err == nil {
		h.Close()
		t.Fatal("accepted unregistered connector")
	}
}
