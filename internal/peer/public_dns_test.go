package peer

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sea-battle-2/internal/game"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestPublicDNSFallbackBoundaries(t *testing.T) {
	for _, host := range []string{"one-two.trycloudflare.com", "abc.trycloudflare.com"} {
		if !quickTunnelHost(host) {
			t.Fatal(host)
		}
	}
	for _, host := range []string{"trycloudflare.com", "a.trycloudflare.com.evil.test", "a.b.trycloudflare.com", "-a.trycloudflare.com", "localhost"} {
		if quickTunnelHost(host) {
			t.Fatal(host)
		}
	}
	queries := 0
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries++
		if r.Header.Get("Accept") != "application/dns-json" || r.URL.Query().Get("name") != "abc.trycloudflare.com" {
			t.Error("unexpected query")
		}
		w.Write([]byte(`{"Status":0,"Answer":[{"name":"abc.trycloudflare.com","type":1,"data":"127.0.0.1"},{"name":"abc.trycloudflare.com","type":1,"data":"10.1.2.3"},{"name":"abc.trycloudflare.com","type":1,"data":"104.16.230.132"}]}`))
	}))
	defer h.Close()
	ips, err := resolvePublicDNS(context.Background(), h.Client(), h.URL, "abc.trycloudflare.com")
	if err != nil || len(ips) != 1 || ips[0].String() != "104.16.230.132" {
		t.Fatal(ips, err)
	}
	if _, err = resolvePublicDNS(context.Background(), h.Client(), h.URL, "private.example"); err == nil {
		t.Fatal("queried arbitrary hostname")
	}
	if queries != 1 {
		t.Fatal("unexpected external DNS request")
	}
}
func TestPublicDNSCannotBypassHTTPSCertificate(t *testing.T) {
	h := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("reached HTTP with untrusted certificate") }))
	defer h.Close()
	_, port, _ := net.SplitHostPort(h.Listener.Addr().String())
	config, _ := websocket.NewConfig("wss://abc.trycloudflare.com:"+port+"/sb2", "https://sea-battle.invalid")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if c, err := websocketAtIP(ctx, config, net.ParseIP("127.0.0.1")); err == nil {
		c.Close()
		t.Fatal("accepted wrong outer certificate")
	}
}

// Explicit opt-in field test: a real host invitation, with only the guest's
// ordinary DNS resolution deliberately broken. Normal test runs stay offline.
func TestFieldPublicDNSFallback(t *testing.T) {
	path := os.Getenv("SB2_FIELD_INVITATION")
	if path == "" {
		t.Skip("set SB2_FIELD_INVITATION for a live cross-machine test")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	invite, err := ParseInvitation(string(b))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	var failedDNS atomic.Int32
	dial := func(ctx context.Context, addr, fp string) (net.Conn, error) {
		if !strings.HasPrefix(addr, "wss://") {
			return dialPinned(ctx, addr, fp)
		}
		config, err := websocket.NewConfig(addr, "https://sea-battle.invalid")
		if err != nil {
			return nil, err
		}
		config.Dialer = &net.Dialer{Resolver: &net.Resolver{PreferGo: true, Dial: func(context.Context, string, string) (net.Conn, error) {
			failedDNS.Add(1)
			return nil, &net.DNSError{Err: "forced field-test DNS failure", Name: config.Location.Hostname(), IsNotFound: true}
		}}}
		ws, err := dialPublicWebSocket(ctx, config)
		if err != nil {
			return nil, err
		}
		ws.PayloadType = websocket.BinaryFrame
		conn := tls.Client(ws, pinnedConfig(fp))
		if err = conn.HandshakeContext(ctx); err != nil {
			abortConn(conn)
			return nil, err
		}
		return conn, nil
	}
	c, err := connectWithDialer(ctx, invite, "", dial)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ready, bootstrap := false, false
	target, last := 0, -1
	for {
		select {
		case <-ctx.Done():
			t.Fatal("field match timed out")
		case m, ok := <-c.Updates:
			if !ok {
				t.Fatal("field connection ended")
			}
			if m.Type == "negotiating" {
				bootstrap = true
			}
			if m.Type == "error" || m.Type == "fatal" {
				t.Fatal(m.Error)
			}
			if m.Type == "offline" || m.Type == "route_failed" {
				t.Log(m.Error)
			}
			v := m.View
			if v == nil {
				continue
			}
			if !ready {
				if m.Transport != "public" || !bootstrap || failedDNS.Load() == 0 {
					t.Fatal("test did not exercise DNS fallback over public transport")
				}
				send(t, c, Command{Type: "ready", Match: v.Match, Fleet: game.RandomFleet(12)})
				ready = true
			}
			if v.Phase == "finished" {
				t.Logf("public DNS fallback: complete match, %d shots, forced DNS failures=%d", len(v.Events), failedDNS.Load())
				return
			}
			if v.Phase == "battle" && v.Turn == v.Player && last != len(v.Events) {
				send(t, c, Command{Type: "fire", Match: v.Match, Expected: len(v.Events), At: game.Coord{X: target % 10, Y: target / 10}})
				last = len(v.Events)
				target++
			}
		}
	}
}

func TestDNSRaceCancelsSlowResolver(t *testing.T) {
	entered := make(chan struct{})
	cancelled := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done(); close(cancelled) }))
	defer slow.Close()
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-entered
		w.Write([]byte(`{"Status":0,"Answer":[{"name":"abc.trycloudflare.com","type":1,"data":"104.16.230.132"}]}`))
	}))
	defer fast.Close()
	ips, err := racePublicDNS(context.Background(), http.DefaultClient, []string{slow.URL, fast.URL}, "abc.trycloudflare.com")
	if err != nil || len(ips) != 1 {
		t.Fatal(ips, err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("losing resolver was not cancelled")
	}
}
