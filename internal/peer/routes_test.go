package peer

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"golang.org/x/net/websocket"
	"net/http/httptest"
	"sea-battle-2/internal/game"
	"strings"
	"testing"
	"time"
)

func TestWebSocketPinnedHandshakeAndShutdown(t *testing.T) {
	s, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	httpServer := httptest.NewServer(s.WebSocketHandler())
	defer httpServer.Close()
	for _, badPin := range []bool{true, false} {
		ws, err := websocket.Dial("ws"+strings.TrimPrefix(httpServer.URL, "http")+"/sb2", "", "http://localhost")
		if err != nil {
			t.Fatal(err)
		}
		ws.PayloadType = websocket.BinaryFrame
		pin := s.LocalInvite().Fingerprint
		if badPin {
			pin = strings.Repeat("0", 64)
		}
		conn := tls.Client(ws, pinnedConfig(pin))
		conn.SetDeadline(time.Now().Add(2 * time.Second))
		err = conn.Handshake()
		if badPin {
			abortConn(conn)
			if err == nil {
				t.Fatal("accepted wrong inner pin")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if err = json.NewEncoder(conn).Encode(Command{Type: "hello", Version: Version, Token: s.LocalInvite().Token}); err != nil {
			t.Fatal(err)
		}
		var m Message
		if err = json.NewDecoder(conn).Decode(&m); err != nil || m.Type != "state" {
			t.Fatalf("handshake: %+v %v", m, err)
		}
		done := make(chan struct{})
		go func() { s.Close(); close(done) }()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("websocket shutdown hung")
		}
		abortConn(conn)
	}
}

func TestExplicitRouteSurvivesHostUpdatesAndReconnect(t *testing.T) {
	s, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	i, _ := s.Invite([]string{"127.0.0.1:1"})
	c, err := ConnectVia(context.Background(), i, s.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	state(t, c, func(v game.View) bool { return v.Connected[1] })
	s.UpdateAddresses([]string{"127.0.0.1:2"})
	// Wait until the actual route update is consumed before breaking the socket.
	deadline := time.Now().Add(time.Second)
	for c.Invitation().Addresses[0] != "127.0.0.1:2" {
		if time.Now().After(deadline) {
			t.Fatal("no route update")
		}
		time.Sleep(time.Millisecond)
	}
	c.mu.Lock()
	abortConn(c.conn)
	c.mu.Unlock()
	state(t, c, func(v game.View) bool { return v.Connected[1] })
}

func TestFailedRoutesReportEndpointAndPin(t *testing.T) {
	s, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	i := s.LocalInvite()
	i.Fingerprint = strings.Repeat("0", 64)
	c := &Client{ctx: context.Background(), invite: i}
	_, err = c.dial()
	if err == nil || !strings.Contains(err.Error(), i.Addresses[0]) || !strings.Contains(err.Error(), "fingerprint mismatch") {
		t.Fatalf("missing diagnostics: %v", err)
	}
	if strings.Contains(err.Error(), i.Token) {
		t.Fatal("credential in diagnostics")
	}
}

func TestTunnelAddressValidation(t *testing.T) {
	for _, a := range []string{"wss://example.com/sb2", "wss://example.com:443/sb2", "wss://[::1]:443/sb2"} {
		if err := ValidAddress(a); err != nil {
			t.Errorf("%s: %v", a, err)
		}
	}
	for _, a := range []string{"ws://example.com/sb2", "wss://user:pass@example.com/sb2", "wss://example.com/wrong", "wss://example.com/sb2?key=secret", "wss://example.com:0/sb2", "wss://0.0.0.0/sb2"} {
		if ValidAddress(a) == nil {
			t.Errorf("accepted %s", a)
		}
	}
}

func TestParallelRoutesFallBackFromWrongHost(t *testing.T) {
	s, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	wrong, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer wrong.Close()
	i, _ := s.Invite([]string{"127.0.0.1:1", wrong.listener.Addr().String(), s.listener.Addr().String()})
	c, err := Connect(context.Background(), i)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	state(t, c, func(v game.View) bool { return v.Connected[1] })
}
