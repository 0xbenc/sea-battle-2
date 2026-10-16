package session

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"sea-battle-2/internal/game"
	"sea-battle-2/internal/netsetup"
	"sea-battle-2/internal/peer"
)

func await(t *testing.T, c *Controller, predicate func(Event) bool) Event {
	t.Helper()
	timer := time.NewTimer(4 * time.Second)
	defer timer.Stop()
	for {
		select {
		case e := <-c.events:
			if e.Generation == c.Generation() && predicate(e) {
				return e
			}
		case <-timer.C:
			t.Fatal("session event timed out", c.State())
		}
	}
}
func TestRestartSerializesCleanupAndRejectsStaleCommands(t *testing.T) {
	c := New(context.Background())
	defer c.Close()
	first := c.Start(Config{Host: true, NoMap: true})
	setup := await(t, c, func(e Event) bool { return e.Port > 0 })
	await(t, c, func(e Event) bool { return e.State == Active })
	second := c.Start(Config{Host: true, NoMap: true, Port: setup.Port})
	if first == second {
		t.Fatal("generation not advanced")
	}
	setup2 := await(t, c, func(e Event) bool { return e.Port > 0 || e.State == Failed })
	if setup2.State == Failed || setup2.Port != setup.Port {
		t.Fatal("replacement bound before prior cleanup", setup2)
	}
	await(t, c, func(e Event) bool { return e.State == Active })
	if err := c.Send(first, peer.Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(1)}); err == nil {
		t.Fatal("stale command reached replacement")
	}
	if err := c.Send(second, peer.Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(1)}); err != nil {
		t.Fatal(err)
	}
	await(t, c, func(e Event) bool { return e.Message.View != nil && e.Message.View.Ready[0] })
	c.Stop()
	await(t, c, func(e Event) bool { return e.State == Stopped })
	l, err := net.Listen("tcp", ":"+strconv.Itoa(setup.Port))
	if err != nil {
		t.Fatal("stop leaked listener", err)
	}
	l.Close()
}
func TestCloseJoinsDiscoveryAndLateMappingCleanup(t *testing.T) {
	c := New(context.Background())
	entered, cleanup, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	c.deps.maintain = func(ctx context.Context, _ int, report func(netsetup.Update)) {
		close(entered)
		<-ctx.Done()
		// Simulate a router returning a lease while cancellation is in flight.
		report(netsetup.Update{State: "active", Address: "203.0.113.1:4747"})
		close(cleanup)
		<-release
	}
	c.Start(Config{Host: true})
	setup := await(t, c, func(e Event) bool { return e.Port > 0 })
	<-entered
	done := make(chan struct{})
	go func() { c.Close(); close(done) }()
	<-cleanup
	select {
	case <-done:
		t.Fatal("Close returned before lease cleanup")
	default:
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close did not join workers")
	}
	if c.State() != Closed {
		t.Fatal(c.State())
	}
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", setup.Port))
	if err != nil {
		t.Fatal(err)
	}
	l.Close()
}
func TestRapidStartStopAndConcurrentClose(t *testing.T) {
	c := New(context.Background())
	for i := 0; i < 40; i++ {
		c.Start(Config{Host: true, NoMap: true})
		if i%2 == 0 {
			c.Stop()
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); c.Close() }()
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown hung")
	}
	gen := c.Generation()
	if c.Start(Config{Host: true}) != gen || c.State() != Closed {
		t.Fatal("closed controller restarted")
	}
	for e := range c.events {
		if e.Generation > gen {
			t.Fatal("impossible generation")
		}
	}
}
func TestRouterUpdatesRefreshInvitationWithoutChangingIdentity(t *testing.T) {
	c := New(context.Background())
	defer c.Close()
	router := make(chan netsetup.Update)
	c.deps.maintain = func(ctx context.Context, _ int, report func(netsetup.Update)) {
		for {
			select {
			case u := <-router:
				report(u)
			case <-ctx.Done():
				return
			}
		}
	}
	c.Start(Config{Host: true})
	setup := await(t, c, func(e Event) bool { return e.Invitation != "" })
	original, err := peer.ParseInvitation(setup.Invitation)
	if err != nil {
		t.Fatal(err)
	}
	guest, err := peer.Connect(context.Background(), original)
	if err != nil {
		t.Fatal(err)
	}
	defer guest.Close()
	await(t, c, func(e Event) bool { return e.Message.View != nil && e.Message.View.Connected[1] })
	router <- netsetup.Update{State: "active", Address: "203.0.113.5:6000", Protocol: "PCP"}
	updated := await(t, c, func(e Event) bool { return e.Invitation != "" && e.Invitation != setup.Invitation })
	invite, err := peer.ParseInvitation(updated.Invitation)
	if err != nil {
		t.Fatal(err)
	}
	if invite.Fingerprint != original.Fingerprint || invite.Token != original.Token || invite.Addresses[0] != "203.0.113.5:6000" {
		t.Fatal("identity or endpoint wrong")
	}
	deadline := time.After(time.Second)
	for guest.Invitation().Addresses[0] != "203.0.113.5:6000" {
		select {
		case <-guest.Updates:
		case <-deadline:
			t.Fatal("guest did not learn route")
		}
	}
	router <- netsetup.Update{State: "unavailable"}
	removed := await(t, c, func(e Event) bool { return e.Invitation != "" && e.Invitation != updated.Invitation })
	cleaned, _ := peer.ParseInvitation(removed.Invitation)
	for _, a := range cleaned.Addresses {
		if a == "203.0.113.5:6000" {
			t.Fatal("expired route still advertised")
		}
	}
}

func TestLocalInterfaceRefreshWithoutRouterMapping(t *testing.T) {
	c := New(context.Background())
	defer c.Close()
	var mu sync.Mutex
	changed := false
	c.deps.refresh = 5 * time.Millisecond
	c.deps.addresses = func(port int) []string {
		mu.Lock()
		defer mu.Unlock()
		host := "127.0.0.1"
		if changed {
			host = "localhost"
		}
		return []string{net.JoinHostPort(host, strconv.Itoa(port))}
	}
	c.deps.maintain = func(context.Context, int, func(netsetup.Update)) { t.Error("--no-map contacted router") }
	c.Start(Config{Host: true, NoMap: true})
	original := await(t, c, func(e Event) bool { return e.Invitation != "" })
	mu.Lock()
	changed = true
	mu.Unlock()
	updated := await(t, c, func(e Event) bool { return e.Invitation != "" && e.Invitation != original.Invitation })
	invite, err := peer.ParseInvitation(updated.Invitation)
	if err != nil {
		t.Fatal(err)
	}
	host, _, _ := net.SplitHostPort(invite.Addresses[0])
	if host != "localhost" {
		t.Fatal("new interface was not advertised")
	}
}
func TestParentCancellationReleasesSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := New(ctx)
	defer c.Close()
	c.Start(Config{Host: true, NoMap: true})
	setup := await(t, c, func(e Event) bool { return e.Port > 0 })
	await(t, c, func(e Event) bool { return e.State == Active })
	c.mu.Lock()
	done := c.tail.done
	c.mu.Unlock()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("parent cancellation did not stop operation")
	}
	if c.State() != Stopped {
		t.Fatal("stale lifecycle after cancellation", c.State())
	}
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", setup.Port))
	if err != nil {
		t.Fatal(err)
	}
	l.Close()
	gen := c.Generation()
	if c.Start(Config{Host: true}) != gen {
		t.Fatal("cancelled controller accepted start")
	}
}

func TestDelayedExpiredRouterResultCannotPublishStaleRoute(t *testing.T) {
	c := New(context.Background())
	defer c.Close()
	c.deps.maintain = func(ctx context.Context, _ int, report func(netsetup.Update)) {
		report(netsetup.Update{State: "active", Address: "203.0.113.99:9999", Expires: time.Now().Add(-time.Second)})
		<-ctx.Done()
	}
	c.Start(Config{Host: true})
	await(t, c, func(e Event) bool {
		if e.Invitation != "" {
			i, err := peer.ParseInvitation(e.Invitation)
			if err != nil {
				t.Fatal(err)
			}
			for _, a := range i.Addresses {
				if a == "203.0.113.99:9999" {
					t.Fatal("expired route was advertised")
				}
			}
		}
		if e.Router != nil {
			if e.Router.State == "active" {
				t.Fatal("late expired grant announced active")
			}
			return e.Router.State == "unavailable"
		}
		return false
	})
}
