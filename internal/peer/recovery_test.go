package peer

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"sea-battle-2/internal/game"
)

func rawSeat(t *testing.T, s *Server, i Invitation) (net.Conn, *json.Decoder) {
	return rawSeatResume(t, s, i, "")
}
func rawSeatResume(t *testing.T, s *Server, i Invitation, resume string) (net.Conn, *json.Decoder) {
	t.Helper()
	c, err := tls.Dial("tcp", s.listener.Addr().String(), pinnedConfig(i.Fingerprint))
	if err != nil {
		t.Fatal(err)
	}
	c.SetDeadline(time.Now().Add(3 * time.Second))
	if err = json.NewEncoder(c).Encode(Command{Type: "hello", Token: i.Token, Version: Version, Resume: resume}); err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(c)
	var m Message
	for {
		if err = dec.Decode(&m); err != nil {
			t.Fatal(err)
		}
		if m.Type == "state" {
			break
		}
	}
	return c, dec
}
func TestShotCommittedWithoutReplyResynchronizesExactlyOnce(t *testing.T) {
	s, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.mu.Lock()
	s.game = game.New(0)
	s.mu.Unlock()
	raw, dec := rawSeat(t, s, s.LocalInvite())
	defer raw.Close()
	invite, _ := s.Invite([]string{s.listener.Addr().String()})
	guest, _ := Connect(context.Background(), invite)
	defer guest.Close()
	state(t, guest, func(v game.View) bool { return v.Connected[0] })
	json.NewEncoder(raw).Encode(Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(1)})
	send(t, guest, Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(2)})
	state(t, guest, func(v game.View) bool { return v.Phase == "battle" })
	var m Message
	for {
		if err := dec.Decode(&m); err != nil {
			t.Fatal(err)
		}
		if m.View != nil && m.View.Phase == "battle" {
			break
		}
	}
	// The host seat stops reading before its shot result. The other player's
	// view proves the command committed; reconnect must recover that single shot.
	json.NewEncoder(raw).Encode(Command{Type: "fire", Match: 1, At: game.Coord{X: 2, Y: 3}})
	state(t, guest, func(v game.View) bool { return len(v.Events) == 1 })
	raw.Close()
	state(t, guest, func(v game.View) bool { return !v.Connected[0] })
	resumed, _ := Connect(context.Background(), s.LocalInvite())
	defer resumed.Close()
	v := state(t, resumed, func(v game.View) bool { return v.Phase == "battle" })
	if len(v.Events) != 1 || v.Events[0].At != (game.Coord{X: 2, Y: 3}) || v.Turn != 1 {
		t.Fatal("lost or duplicated interrupted shot")
	}
	send(t, guest, Command{Type: "fire", Match: 1, Expected: 1, At: game.Coord{X: 4, Y: 5}})
	v = state(t, resumed, func(v game.View) bool { return len(v.Events) == 2 })
	if v.Events[0].At != (game.Coord{X: 2, Y: 3}) {
		t.Fatal("history changed")
	}
}
func TestEndpointUpdatesAndRepeatedReconnectResetBackoff(t *testing.T) {
	s, a, b, _ := setup(t)
	for n := 0; n < 4; n++ {
		// Failed route first; the reachable candidate must still win.
		i, err := s.UpdateAddresses([]string{"127.0.0.1:1", s.listener.Addr().String()})
		if err != nil {
			t.Fatal(err)
		}
		timer := time.After(time.Second)
		for b.Invitation().Addresses[0] != i.Addresses[0] {
			select {
			case <-b.Updates:
			case <-timer:
				t.Fatal("route not received")
			}
		}
		b.mu.Lock()
		b.conn.Close()
		b.mu.Unlock()
		state(t, a, func(v game.View) bool { return !v.Connected[1] })
		sawOffline := false
		deadline := time.After(3 * time.Second)
		for {
			select {
			case m := <-b.Updates:
				if m.Type == "offline" {
					sawOffline = true
					if !strings.Contains(m.Error, "retrying in 1s") {
						t.Fatalf("successful connection did not reset backoff: %s", m.Error)
					}
				}
				if sawOffline && m.View != nil && m.View.Connected[1] {
					goto rejoined
				}
			case <-deadline:
				t.Fatal("reconnect exceeded reset backoff")
			}
		}
	rejoined:
		state(t, a, func(v game.View) bool { return v.Connected[1] })
	}
	wrong := b.Invitation()
	wrong.Fingerprint = strings.Repeat("0", 64)
	if b.UpdateInvitation(wrong) == nil {
		t.Fatal("route update changed host identity")
	}
}
func TestCloseInterruptsUnauthenticatedSockets(t *testing.T) {
	s, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 16; n++ {
		c, err := net.Dial("tcp", s.listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		// Send an incomplete TLS handshake. No peer seat is authenticated.
		c.Write([]byte{0x16})
	}
	done := make(chan struct{})
	go func() { s.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown waited for handshake deadlines")
	}
}
func TestSlowConsumerDisconnectsWithoutBlockingOtherSeat(t *testing.T) {
	s, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	serverConn, slow := net.Pipe()
	defer slow.Close()
	// Drive the actual server read/write loops with a transport whose blocked
	// writes are deterministic. The other seat uses real pinned TLS.
	s.mu.Lock()
	s.connections[serverConn] = struct{}{}
	s.mu.Unlock()
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() { s.mu.Lock(); delete(s.connections, serverConn); s.mu.Unlock() }()
		s.serve(serverConn)
	}()
	enc := json.NewEncoder(slow)
	enc.Encode(Command{Type: "hello", Token: s.LocalInvite().Token, Version: Version})
	var initial Message
	if err := json.NewDecoder(slow).Decode(&initial); err != nil {
		t.Fatal(err)
	}
	invite, _ := s.Invite([]string{s.listener.Addr().String()})
	guest, err := Connect(context.Background(), invite)
	if err != nil {
		t.Fatal(err)
	}
	defer guest.Close()
	state(t, guest, func(v game.View) bool { return v.Connected[0] && v.Connected[1] })
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for i := 0; i < 30; i++ {
			if enc.Encode(Command{Type: "ping"}) != nil {
				return
			}
		}
	}()
	// The slow seat keeps writing but no longer reads a single response.
	state(t, guest, func(v game.View) bool { return !v.Connected[0] })
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("slow peer writer stuck")
	}
	send(t, guest, Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(3)})
	state(t, guest, func(v game.View) bool { return v.Ready[1] })
}
func TestOversizedAndTruncatedCommandsCannotMutateMatch(t *testing.T) {
	for _, payload := range []string{strings.Repeat("x", maxMessage+1) + "\n", `{"type":"fire","at":`} {
		t.Run(string(rune(len(payload)%26+'a')), func(t *testing.T) {
			s, err := Host("127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			c, _ := rawSeat(t, s, s.LocalInvite())
			c.Write([]byte(payload))
			if len(payload) < maxMessage {
				c.Close()
			} else {
				c.SetReadDeadline(time.Now().Add(time.Second))
				scanner := bufio.NewScanner(c)
				if scanner.Scan() {
					t.Fatal("oversized command received a normal response")
				}
				c.Close()
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			if len(s.game.Events) != 0 || s.game.Ready[0] {
				t.Fatal("invalid wire input mutated game")
			}
		})
	}
}
func TestCloseInterruptsBlockedWrite(t *testing.T) {
	reader, writer := net.Pipe()
	defer reader.Close()
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{ctx: ctx, cancel: cancel, conn: writer, ready: true, done: make(chan struct{})}
	writing := make(chan struct{})
	go func() { defer close(writing); c.Send(Command{Type: "ping"}) }()
	// A reader establishes that a write began, then deliberately stops draining.
	b := make([]byte, 1)
	reader.Read(b)
	close(c.done)
	done := make(chan struct{})
	go func() { c.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close blocked behind write mutex")
	}
	<-writing
}

func TestStressRepeatedMatchesAndRematches(t *testing.T) {
	_, a, b, _ := setup(t)
	players := [2]*Client{a, b}
	priorStarter := -1
	for match := 1; match <= 20; match++ {
		fleets := [2][]game.Placement{game.RandomFleet(uint64(match * 2)), game.RandomFleet(uint64(match*2 + 1))}
		for p, c := range players {
			send(t, c, Command{Type: "ready", Match: match, Fleet: fleets[p]})
		}
		v := state(t, a, func(v game.View) bool { return v.Match == match && v.Phase == "battle" })
		state(t, b, func(v game.View) bool { return v.Match == match && v.Phase == "battle" })
		starter := v.Turn
		if starter == priorStarter {
			t.Fatal("rematch did not alternate starter")
		}
		priorStarter = starter
		var targets [2][]game.Coord
		for p := 0; p < 2; p++ {
			for i, s := range fleets[1-p] {
				targets[p] = append(targets[p], game.Cells(s, game.Lengths[i])...)
			}
		}
		next := [2]int{}
		for v.Phase == "battle" {
			p := v.Turn
			at := targets[p][next[p]]
			next[p]++
			sequence := len(v.Events) + 1
			send(t, players[p], Command{Type: "fire", Match: match, Expected: sequence - 1, At: at})
			v = state(t, a, func(v game.View) bool { return len(v.Events) == sequence })
			vb := state(t, b, func(v game.View) bool { return len(v.Events) == sequence })
			if v.Phase != vb.Phase || v.Winner != vb.Winner {
				t.Fatal("clients disagree")
			}
		}
		if v.Winner != starter {
			t.Fatal("wrong winner in all-hit race")
		}
		for _, c := range players {
			send(t, c, Command{Type: "rematch", Match: match})
		}
		v = state(t, a, func(v game.View) bool { return v.Match == match+1 })
		state(t, b, func(v game.View) bool { return v.Match == match+1 })
		if len(v.Events) != 0 || len(v.Own) != 0 || v.Ready[0] || v.Ready[1] {
			t.Fatal("previous match leaked into rematch")
		}
	}
}

func TestRepeatedShotCommandCommitsOnlyOnce(t *testing.T) {
	s, a, b, _ := setup(t)
	send(t, a, Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(1)})
	send(t, b, Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(2)})
	v := state(t, a, func(v game.View) bool { return v.Phase == "battle" })
	state(t, b, func(v game.View) bool { return v.Phase == "battle" })
	shooter := a
	if v.Turn == 1 {
		shooter = b
	}
	cmd := Command{Type: "fire", Match: 1, At: game.Coord{X: 1, Y: 1}}
	for i := 0; i < 3; i++ {
		send(t, shooter, cmd)
	}
	// Drain the shooter's results/errors until both duplicates have been rejected.
	errorsSeen := 0
	deadline := time.After(time.Second)
	for errorsSeen < 2 {
		select {
		case m := <-shooter.Updates:
			if m.Type == "error" {
				errorsSeen++
			}
		case <-deadline:
			t.Fatal("duplicates not rejected")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.game.Events) != 1 {
		t.Fatal("repeated input consumed multiple turns")
	}
}

func TestPastedRouteUpdateRecoversCompletelyBrokenPath(t *testing.T) {
	_, a, b, original := setup(t)
	broken := original
	broken.Addresses = []string{"127.0.0.1:1"}
	if err := b.UpdateInvitation(broken); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	b.conn.Close()
	b.mu.Unlock()
	state(t, a, func(v game.View) bool { return !v.Connected[1] })
	deadline := time.After(3 * time.Second)
	for {
		select {
		case m := <-b.Updates:
			if m.Type == "offline" {
				goto offline
			}
		case <-deadline:
			t.Fatal("client did not detect lost path")
		}
	}
offline:
	if err := b.UpdateInvitation(original); err != nil {
		t.Fatal(err)
	}
	state(t, b, func(v game.View) bool { return v.Connected[0] && v.Connected[1] })
}

func TestQueuedIntentCannotCrossReconnect(t *testing.T) {
	s, a, b, _ := setup(t)
	send(t, a, Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(1)})
	send(t, b, Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(2)})
	v := state(t, a, func(v game.View) bool { return v.Phase == "battle" })
	state(t, b, func(v game.View) bool { return v.Phase == "battle" })
	shooter, other := a, b
	if v.Turn == 1 {
		shooter, other = b, a
	}
	execute := shooter.Prepare(Command{Type: "fire", Match: 1, At: game.Coord{X: 3, Y: 3}})
	shooter.mu.Lock()
	shooter.conn.Close()
	shooter.mu.Unlock()
	player := v.Turn
	state(t, other, func(v game.View) bool { return !v.Connected[player] })
	state(t, shooter, func(v game.View) bool { return v.Connected[0] && v.Connected[1] })
	if execute() == nil {
		t.Fatal("queued intent migrated across reconnect")
	}
	s.mu.Lock()
	events := len(s.game.Events)
	s.mu.Unlock()
	if events != 0 {
		t.Fatal("stale intent fired")
	}
	send(t, shooter, Command{Type: "fire", Match: 1, At: game.Coord{X: 3, Y: 3}})
	state(t, shooter, func(v game.View) bool { return len(v.Events) == 1 })
}

func TestSameClientResumeReplacesHalfOpenSeat(t *testing.T) {
	s, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	resume, _ := secret()
	old, _ := rawSeatResume(t, s, s.LocalInvite(), resume)
	defer old.Close()
	fresh, dec := rawSeatResume(t, s, s.LocalInvite(), resume)
	defer fresh.Close()
	// No FIN or timeout on the old socket was needed to resume this seat.
	old.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := old.Read(make([]byte, 1)); err == nil {
		t.Fatal("old transport remained live")
	}
	json.NewEncoder(fresh).Encode(Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(8)})
	var m Message
	for {
		if err := dec.Decode(&m); err != nil {
			t.Fatal(err)
		}
		if m.View != nil && m.View.Ready[0] {
			break
		}
	}
	s.mu.Lock()
	peers := s.peers[0] != nil
	s.mu.Unlock()
	if !peers {
		t.Fatal("old connection cleanup removed replacement seat")
	}
}

func TestCancellationInterruptsInFlightTLSHandshake(t *testing.T) {
	s, err := Host("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	invite := s.LocalInvite()
	s.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	invite.Addresses = []string{listener.Addr().String()}
	entered, serverDone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		if _, err := io.ReadFull(conn, make([]byte, 5)); err != nil {
			return
		}
		close(entered)
		io.Copy(io.Discard, conn) // Never reply to the TLS ClientHello.
	}()
	c, err := Connect(context.Background(), invite)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("TLS handshake did not start")
	}
	done := make(chan struct{})
	go func() { c.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancel did not interrupt TLS handshake")
	}
	select {
	case <-serverDone:
	case <-time.After(time.Second):
		t.Fatal("abandoned handshake socket leaked")
	}
}
