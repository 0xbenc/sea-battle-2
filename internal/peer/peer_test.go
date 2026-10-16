package peer

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"sea-battle-2/internal/game"
)

func state(t *testing.T, c *Client, predicate func(game.View) bool) game.View {
	t.Helper()
	timer := time.NewTimer(4 * time.Second)
	defer timer.Stop()
	for {
		select {
		case m, ok := <-c.Updates:
			if !ok {
				t.Fatal("client stopped")
			}
			if m.Type == "error" || m.Type == "fatal" {
				t.Fatal(m.Error)
			}
			if m.View != nil && predicate(*m.View) {
				return *m.View
			}
		case <-timer.C:
			t.Fatal("state timed out")
		}
	}
}
func send(t *testing.T, c *Client, cmd Command) {
	t.Helper()
	if err := c.Send(cmd); err != nil {
		t.Fatal(err)
	}
}
func setup(t *testing.T) (*Server, *Client, *Client, Invitation) {
	t.Helper()
	s, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	invite, err := s.Invite([]string{s.listener.Addr().String()})
	if err != nil {
		t.Fatal(err)
	}
	a, err := Connect(context.Background(), s.LocalInvite())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Close)
	b, err := Connect(context.Background(), invite)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.Close)
	state(t, a, func(v game.View) bool { return v.Connected[0] && v.Connected[1] })
	state(t, b, func(v game.View) bool { return v.Connected[0] && v.Connected[1] })
	return s, a, b, invite
}
func TestCompletePrivateMatchOverTLS(t *testing.T) {
	_, a, b, _ := setup(t)
	fa, fb := game.RandomFleet(1), game.RandomFleet(2)
	send(t, a, Command{Type: "ready", Fleet: fa, Match: 1})
	send(t, b, Command{Type: "ready", Fleet: fb, Match: 1})
	va := state(t, a, func(v game.View) bool { return v.Phase == "battle" })
	vb := state(t, b, func(v game.View) bool { return v.Phase == "battle" })
	if len(va.Own) != 5 || len(vb.Own) != 5 {
		t.Fatal("missing fleet")
	}
	// Give A all 17 enemy cells and B only misses. The transport still must
	// enforce the random starting turn and conceal every unsunk hit identity.
	var targets [2][]game.Coord
	for i, p := range fb {
		targets[0] = append(targets[0], game.Cells(p, game.Lengths[i])...)
	}
	occupied := [100]bool{}
	for i, p := range fa {
		for _, c := range game.Cells(p, game.Lengths[i]) {
			occupied[c.Index()] = true
		}
	}
	for i, yes := range occupied {
		if !yes {
			targets[1] = append(targets[1], game.Coord{X: i % 10, Y: i / 10})
		}
	}
	clients := [2]*Client{a, b}
	indices := [2]int{}
	for va.Phase != "finished" {
		p := va.Turn
		at := targets[p][indices[p]]
		indices[p]++
		n := len(va.Events) + 1
		send(t, clients[p], Command{Type: "fire", At: at, Expected: n - 1, Match: 1})
		va = state(t, a, func(v game.View) bool { return len(v.Events) == n })
		vb = state(t, b, func(v game.View) bool { return len(v.Events) == n })
		e := va.Events[n-1]
		if e.Player == 0 && e.Hit && !e.Sunk && e.Ship != -1 {
			t.Fatal("enemy identity leaked")
		}
		if va.Turn != vb.Turn || va.Phase != vb.Phase {
			t.Fatal("desynchronized")
		}
	}
	if va.Winner != 0 || vb.Winner != 0 {
		t.Fatal("wrong winner")
	}
	send(t, a, Command{Type: "rematch", Match: 1})
	send(t, b, Command{Type: "rematch", Match: 1})
	v := state(t, a, func(v game.View) bool { return v.Match == 2 })
	if v.Phase != "placement" || len(v.Events) != 0 || len(v.Own) != 0 {
		t.Fatal("dirty rematch")
	}
}
func TestReconnectPreservesFleetAndPausesFire(t *testing.T) {
	s, a, b, _ := setup(t)
	send(t, a, Command{Type: "ready", Fleet: game.RandomFleet(3), Match: 1})
	send(t, b, Command{Type: "ready", Fleet: game.RandomFleet(4), Match: 1})
	va := state(t, a, func(v game.View) bool { return v.Phase == "battle" })
	state(t, b, func(v game.View) bool { return v.Phase == "battle" })
	// A forced socket interruption exercises automatic reconnect, not a new identity.
	b.mu.Lock()
	b.conn.Close()
	b.mu.Unlock()
	state(t, a, func(v game.View) bool { return !v.Connected[1] })
	if va.Turn == 0 {
		send(t, a, Command{Type: "fire", At: game.Coord{}, Match: 1})
		select {
		case m := <-a.Updates:
			if m.Type != "error" {
				t.Fatalf("expected pause rejection, got %s", m.Type)
			}
		case <-time.After(time.Second):
			t.Fatal("no rejection")
		}
	}
	v := state(t, b, func(v game.View) bool { return v.Phase == "battle" && v.Ready[1] })
	if len(v.Own) != 5 || len(v.Events) != 0 {
		t.Fatal("reconnect lost state")
	}
	s.mu.Lock()
	if len(s.game.Events) != 0 {
		t.Error("shot occurred while offline")
	}
	s.mu.Unlock()
}
func TestInvitationAndPin(t *testing.T) {
	s, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	i := s.LocalInvite()
	code, _ := i.Encode()
	parsed, err := ParseInvitation(code[:20] + "\n" + code[20:])
	if err != nil || parsed.Token != i.Token {
		t.Fatal("wrapped invitation")
	}
	bad := i
	bad.Fingerprint = strings.Repeat("0", 64)
	d := tls.Dialer{Config: pinnedConfig(bad.Fingerprint)}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	c, err := d.DialContext(ctx, "tcp", i.Addresses[0])
	if err == nil {
		c.Close()
		t.Fatal("accepted wrong host pin")
	}
	bad = i
	bad.Expires = time.Now().Add(-time.Second).Unix()
	if _, err = bad.Encode(); err == nil {
		t.Fatal("expired invite")
	}
	for _, a := range []string{"0.0.0.0:4", "[::]:4", "host:0", "host:65536", "bad host:5", "224.0.0.1:5"} {
		if ValidAddress(a) == nil {
			t.Errorf("accepted %s", a)
		}
	}
	if _, err = ParseInvitation(strings.Repeat("a", MaxInvitation+1)); err == nil {
		t.Fatal("unbounded invite")
	}
}
func TestAuthenticationAndDuplicateSeat(t *testing.T) {
	s, _, _, i := setup(t)
	dial := func(token string) net.Conn {
		t.Helper()
		c, err := tls.Dial("tcp", s.listener.Addr().String(), pinnedConfig(i.Fingerprint))
		if err != nil {
			t.Fatal(err)
		}
		c.SetDeadline(time.Now().Add(time.Second))
		json.NewEncoder(c).Encode(Command{Type: "hello", Version: Version, Token: token})
		return c
	}
	c := dial("incorrect")
	var m Message
	if err := json.NewDecoder(c).Decode(&m); err == nil && m.Type != "fatal" {
		t.Fatal("bad token accepted")
	}
	c.Close()
	c = dial(i.Token)
	defer c.Close()
	if err := json.NewDecoder(c).Decode(&m); err != nil || m.Type != "error" {
		t.Fatalf("duplicate seat: %+v %v", m, err)
	}
}
func TestClientCancelDuringUnreachableDial(t *testing.T) {
	s, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	i := s.LocalInvite()
	s.Close()
	i.Addresses = []string{"192.0.2.1:4747", "198.51.100.1:4747"}
	c, err := Connect(context.Background(), i)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { c.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancel did not stop dialing")
	}
}
