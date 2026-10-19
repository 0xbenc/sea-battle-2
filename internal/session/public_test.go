package session

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"sea-battle-2/internal/peer"
	"sea-battle-2/internal/tunnel"
)

type fakeTunnel struct {
	address string
	done    chan struct{}
	once    sync.Once
}

func (t *fakeTunnel) Address() string       { return t.address }
func (t *fakeTunnel) Done() <-chan struct{} { return t.done }
func (t *fakeTunnel) Close()                { t.once.Do(func() { close(t.done) }) }

func TestPublicLifecycleRetiresOnlyConfirmedDirectConnection(t *testing.T) {
	c := New(context.Background())
	defer c.Close()
	connector := &fakeTunnel{address: "wss://127.0.0.1:1/sb2", done: make(chan struct{})}
	port := make(chan int, 1)
	c.deps.tunnel = func(ctx context.Context, p int, report func(string)) (tunnel.Handle, error) {
		port <- p
		return connector, nil
	}
	c.Start(Config{Host: true, NoMap: true, AutoPublic: true})
	initial := await(t, c, func(e Event) bool { return e.Port > 0 })
	if initial.Invitation != "" {
		t.Fatal("shared an incomplete invitation before public setup")
	}
	setup := await(t, c, func(e Event) bool { return e.Invitation != "" })
	invite, err := peer.ParseInvitation(setup.Invitation)
	if err != nil {
		t.Fatal(err)
	}
	if invite.Addresses[0] != connector.address {
		t.Fatal("missing public route")
	}
	select {
	case <-connector.Done():
		t.Fatal("retired before guest arrived")
	default:
	}
	guest, err := peer.Connect(context.Background(), invite)
	if err != nil {
		t.Fatal(err)
	}
	defer guest.Close()
	// The fake public endpoint refuses connections. The guest's direct fallback
	// must still confirm a working session before its connector is retired.
	await(t, c, func(e Event) bool { return e.Network == "direct" })
	select {
	case <-connector.Done():
	default:
		t.Fatal("direct session retained public process")
	}
	deadline := time.Now().Add(time.Second)
	for strings.HasPrefix(guest.Invitation().Addresses[0], "wss://") {
		if time.Now().After(deadline) {
			t.Fatal("retired route still advertised")
		}
		time.Sleep(time.Millisecond)
	}
	c.Close()
	p := <-port
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(p)))
	if err != nil {
		t.Fatal("HTTP listener leaked", err)
	}
	l.Close()
}

func TestPublicSetupCancellationJoinsWorker(t *testing.T) {
	c := New(context.Background())
	entered := make(chan struct{})
	exited := make(chan struct{})
	c.deps.tunnel = func(ctx context.Context, p int, report func(string)) (tunnel.Handle, error) {
		close(entered)
		<-ctx.Done()
		close(exited)
		return nil, ctx.Err()
	}
	c.Start(Config{Host: true, NoMap: true, AutoPublic: true})
	<-entered
	c.Close()
	select {
	case <-exited:
	default:
		t.Fatal("connector setup outlived session")
	}
}

func TestPublicSetupFailureStillSharesDirectInvitation(t *testing.T) {
	c := New(context.Background())
	defer c.Close()
	c.deps.tunnel = func(context.Context, int, func(string)) (tunnel.Handle, error) { return nil, errors.New("offline") }
	c.Start(Config{Host: true, NoMap: true, AutoPublic: true})
	await(t, c, func(e Event) bool { return e.Network == "unavailable" })
	e := await(t, c, func(e Event) bool { return e.Invitation != "" })
	i, err := peer.ParseInvitation(e.Invitation)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range i.Addresses {
		if strings.HasPrefix(a, "wss://") {
			t.Fatal("failed route advertised")
		}
	}
}
