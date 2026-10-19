package tunnel

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sea-battle-2/internal/peer"
	"strings"
	"sync"
	"time"
)

// Handle owns exactly one connector process. Close is idempotent and joins it.
type Handle interface {
	Address() string
	Done() <-chan struct{}
	Close()
}
type process struct {
	address string
	cancel  context.CancelFunc
	done    chan struct{}
}

func (p *process) Address() string       { return p.address }
func (p *process) Done() <-chan struct{} { return p.done }
func (p *process) Close()                { p.cancel(); <-p.done }

func Start(ctx context.Context, port int, report func(string)) (Handle, error) {
	setup, stop := context.WithTimeout(ctx, 60*time.Second)
	defer stop()
	path, err := helper(setup, report)
	if err != nil {
		return nil, err
	}
	report("Opening a secure public connection…")
	h, err := launch(ctx, path, port, 35*time.Second)
	if err != nil {
		return nil, err
	}
	report("Verifying the public invitation…")
	ready, stopReady := context.WithTimeout(ctx, 8*time.Second)
	defer stopReady()
	for {
		attempt, stopAttempt := context.WithTimeout(ready, 4*time.Second)
		err = peer.CheckPublicEndpoint(attempt, h.Address())
		stopAttempt()
		if err == nil {
			return h, nil
		}
		select {
		case <-ready.Done():
			if ctx.Err() != nil {
				h.Close()
				return nil, ctx.Err()
			}
			// A host's own egress/DNS is not proof of guest reachability. Keep the
			// registered connector available; the guest verifies the pinned route.
			report("Invitation prepared. Your friend will check the available routes.")
			return h, nil
		case <-h.Done():
			h.Close()
			return nil, errors.New("public connector exited during verification")
		case <-time.After(500 * time.Millisecond):
		}
	}
}

var publicURL = regexp.MustCompile(`https://[a-z0-9]+(?:-[a-z0-9]+)*\.trycloudflare\.com`)

func launch(ctx context.Context, path string, port int, timeout time.Duration) (Handle, error) {
	ctx, cancel := context.WithCancel(ctx)
	p := &process{cancel: cancel, done: make(chan struct{})}
	dir, err := os.MkdirTemp("", "sea-battle-tunnel-*")
	if err != nil {
		cancel()
		return nil, err
	}
	// An empty explicit configuration avoids using unrelated account/tunnel settings.
	config := filepath.Join(dir, "config.yml")
	if err = os.WriteFile(config, []byte("{}\n"), 0600); err != nil {
		os.RemoveAll(dir)
		cancel()
		return nil, err
	}
	cmd := exec.CommandContext(ctx, path, "--config", config, "tunnel", "--url", fmt.Sprintf("http://127.0.0.1:%d", port), "--no-autoupdate")
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "TUNNEL_") {
			cmd.Env = append(cmd.Env, e)
		}
	}
	cmd.Cancel = func() error {
		if runtime.GOOS == "windows" {
			return cmd.Process.Kill()
		}
		return cmd.Process.Signal(os.Interrupt)
	}
	cmd.WaitDelay = 2 * time.Second
	reader, writer := io.Pipe()
	cmd.Stdout = writer
	cmd.Stderr = writer
	ready := make(chan string, 1)
	var scanWG sync.WaitGroup
	scanWG.Add(1)
	go func() {
		defer scanWG.Done()
		defer reader.Close()
		sc := bufio.NewScanner(reader)
		sc.Buffer(make([]byte, 4096), 64<<10)
		address := ""
		registered := false
		for sc.Scan() {
			line := sc.Text()
			if u := publicURL.FindString(line); u != "" {
				address = strings.Replace(u, "https://", "wss://", 1) + "/sb2"
			}
			if strings.Contains(line, "Registered tunnel connection") {
				registered = true
			}
			if address != "" && registered {
				select {
				case ready <- address:
				default:
				}
			}
		}
	}()
	if err = cmd.Start(); err != nil {
		writer.Close()
		scanWG.Wait()
		os.RemoveAll(dir)
		cancel()
		return nil, err
	}
	go func() { cmd.Wait(); writer.Close(); scanWG.Wait(); os.RemoveAll(dir); close(p.done) }()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case p.address = <-ready:
		return p, nil
	case <-p.done:
		cancel()
		return nil, errors.New("public connector exited before it was ready")
	case <-timer.C:
		p.Close()
		return nil, errors.New("public connection setup timed out")
	case <-ctx.Done():
		p.Close()
		return nil, ctx.Err()
	}
}
