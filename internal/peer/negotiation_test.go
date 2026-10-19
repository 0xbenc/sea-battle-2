package peer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"sea-battle-2/internal/game"
)

func publicFixture(t *testing.T, direct bool) (*Server, Invitation, func(context.Context, string, string) (net.Conn, error), *atomic.Int32) {
	t.Helper()
	s, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	h := httptest.NewServer(s.WebSocketHandler())
	t.Cleanup(h.Close)
	addresses := []string{"wss://test.invalid/sb2"}
	if direct {
		addresses = append(addresses, s.listener.Addr().String())
	} else {
		addresses = append(addresses, "127.0.0.1:1")
	}
	i, err := s.UpdateAddresses(addresses)
	if err != nil {
		t.Fatal(err)
	}
	calls := new(atomic.Int32)
	dial := func(ctx context.Context, addr, fp string) (net.Conn, error) {
		if strings.HasPrefix(addr, "wss://") {
			calls.Add(1)
			return dialWebSocket(ctx, "ws"+strings.TrimPrefix(h.URL, "http")+"/sb2", fp)
		}
		return dialPinned(ctx, addr, fp)
	}
	return s, i, dial, calls
}

func TestAutomaticPublicBootstrapAndDirectHandoff(t *testing.T) {
	s, i, dial, publicCalls := publicFixture(t, true)
	host, err := Connect(context.Background(), s.LocalInvite())
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	guest, err := connectWithDialer(context.Background(), i, "", dial)
	if err != nil {
		t.Fatal(err)
	}
	defer guest.Close()
	awaitBootstrap(t, guest)
	state(t, guest, func(v game.View) bool { return v.Connected[0] && v.Connected[1] })
	mode, confirmed := s.GuestLink()
	if mode != "direct" || !confirmed || publicCalls.Load() != 1 {
		t.Fatalf("link=%s confirmed=%v public calls=%d", mode, confirmed, publicCalls.Load())
	}
	send(t, host, Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(1)})
	send(t, guest, Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(2)})
	v := state(t, guest, func(v game.View) bool { return v.Phase == "battle" })
	clients := [2]*Client{host, guest}
	send(t, clients[v.Turn], Command{Type: "fire", Match: 1, At: game.Coord{X: 3, Y: 4}})
	state(t, guest, func(v game.View) bool { return len(v.Events) == 1 })
}

func TestUnreachableDirectKeepsPublicSession(t *testing.T) {
	s, i, dial, _ := publicFixture(t, false)
	guest, err := connectWithDialer(context.Background(), i, "", dial)
	if err != nil {
		t.Fatal(err)
	}
	defer guest.Close()
	state(t, guest, func(v game.View) bool { return v.Connected[1] })
	mode, confirmed := s.GuestLink()
	if mode != "public" || !confirmed {
		t.Fatalf("%s %v", mode, confirmed)
	}
	send(t, guest, Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(7)})
	state(t, guest, func(v game.View) bool { return v.Ready[1] })
}

func TestPublicUnavailableFallsBackToDirect(t *testing.T) {
	s, i, dial, _ := publicFixture(t, true)
	guest, err := connectWithDialer(context.Background(), i, "", func(ctx context.Context, a, fp string) (net.Conn, error) {
		if strings.HasPrefix(a, "wss://") {
			return nil, errors.New("public service unavailable")
		}
		return dial(ctx, a, fp)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer guest.Close()
	state(t, guest, func(v game.View) bool { return v.Connected[1] })
	mode, ok := s.GuestLink()
	if mode != "direct" || !ok {
		t.Fatal(mode, ok)
	}
}

func TestRouteProbeCannotDisplaceOccupiedSeat(t *testing.T) {
	s, _, guest, i := setup(t)
	conn, err := dialPinned(context.Background(), s.listener.Addr().String(), i.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	defer abortConn(conn)
	conn.SetDeadline(time.Now().Add(time.Second))
	json.NewEncoder(conn).Encode(Command{Type: "probe", Token: i.Token, Version: Version, Resume: "another-client"})
	if _, err := scanMessage(scanner(conn)); err == nil {
		t.Fatal("independent client authorized a handoff")
	}
	send(t, guest, Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(5)})
	state(t, guest, func(v game.View) bool { return v.Ready[1] })
}

func TestCancelledNegotiationJoinsProbeWorkers(t *testing.T) {
	_, i, dial, _ := publicFixture(t, true)
	entered := make(chan struct{})
	guest, err := connectWithDialer(context.Background(), i, "", func(ctx context.Context, a, fp string) (net.Conn, error) {
		if !strings.HasPrefix(a, "wss://") {
			close(entered)
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return dial(ctx, a, fp)
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		guest.Close()
		t.Fatal("probe did not start")
	}
	done := make(chan struct{})
	go func() { guest.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("close did not join negotiation")
	}
}

func awaitBootstrap(t *testing.T, c *Client) {
	t.Helper()
	timeout := time.After(time.Second)
	for {
		select {
		case m := <-c.Updates:
			if m.Type == "negotiating" {
				return
			}
			if m.Type == "state" {
				t.Fatal("game state exposed before public negotiation")
			}
		case <-timeout:
			t.Fatal("no public bootstrap")
		}
	}
}

type failActivationConn struct{ net.Conn }

func (c failActivationConn) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte(`"type":"hello"`)) {
		return 0, io.ErrClosedPipe
	}
	return c.Conn.Write(p)
}
func TestFailedActivationRetainsWorkingPublicSocket(t *testing.T) {
	s, i, dial, calls := publicFixture(t, true)
	guest, err := connectWithDialer(context.Background(), i, "", func(ctx context.Context, a, fp string) (net.Conn, error) {
		conn, err := dial(ctx, a, fp)
		if err == nil && !strings.HasPrefix(a, "wss://") {
			return failActivationConn{conn}, nil
		}
		return conn, err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer guest.Close()
	awaitBootstrap(t, guest)
	state(t, guest, func(v game.View) bool { return v.Connected[1] })
	mode, confirmed := s.GuestLink()
	if mode != "public" || !confirmed || calls.Load() != 1 {
		t.Fatalf("lost public session: %s %v calls=%d", mode, confirmed, calls.Load())
	}
}
