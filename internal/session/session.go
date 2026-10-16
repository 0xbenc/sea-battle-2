// Package session owns the entire lifetime of local and direct multiplayer sessions.
// Presentation consumes generation-tagged events; it never owns network resources.
package session

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"sea-battle-2/internal/game"
	"sea-battle-2/internal/netsetup"
	"sea-battle-2/internal/peer"
)

type State string

const (
	Idle         State = "idle"
	Starting     State = "starting"
	Connecting   State = "connecting"
	Active       State = "active"
	Reconnecting State = "reconnecting"
	Failed       State = "failed"
	Stopping     State = "stopping"
	Stopped      State = "stopped"
	Closed       State = "closed"
)

type Config struct {
	Solo            bool
	Human           int
	Host            bool
	Join, Advertise string
	Port            int
	NoMap           bool
}
type Event struct {
	Generation uint64
	State      State
	Message    peer.Message
	Solo       bool
	Host       bool
	Port       int
	Invitation string
	Router     *netsetup.Update
	Error      string
}
type operation struct {
	generation uint64
	ctx        context.Context
	cancel     context.CancelFunc
	done       chan struct{}
	client     *peer.Client      // guarded by Controller.mu
	local      chan localCommand // installed under Controller.mu before publication
}
type dependencies struct {
	host      func(string) (*peer.Server, error)
	connect   func(context.Context, peer.Invitation) (*peer.Client, error)
	maintain  func(context.Context, int, func(netsetup.Update))
	addresses func(int) []string
	refresh   time.Duration
	botDelay  time.Duration
}
type Controller struct {
	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	events     chan Event
	tail       *operation
	generation uint64
	state      State
	closed     bool
	wg         sync.WaitGroup
	once       sync.Once
	deps       dependencies
}

func New(parent context.Context) *Controller {
	ctx, cancel := context.WithCancel(parent)
	return &Controller{ctx: ctx, cancel: cancel, events: make(chan Event, 64), state: Idle,
		deps: dependencies{host: peer.Host, connect: peer.Connect, maintain: netsetup.Maintain, addresses: peer.LocalAddresses, refresh: 10 * time.Second, botDelay: 650 * time.Millisecond}}
}
func (c *Controller) Generation() uint64 { c.mu.Lock(); defer c.mu.Unlock(); return c.generation }
func (c *Controller) State() State       { c.mu.Lock(); defer c.mu.Unlock(); return c.state }
func (c *Controller) Next() (Event, bool) {
	select {
	case e, ok := <-c.events:
		return e, ok
	case <-c.ctx.Done():
		return Event{}, false
	}
}
func (c *Controller) Start(config Config) uint64 { return c.start(config, false) }
func (c *Controller) Stop() uint64               { return c.start(Config{}, true) }
func (c *Controller) start(config Config, stop bool) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.ctx.Err() != nil {
		return c.generation
	}
	previous := c.tail
	if previous != nil {
		previous.cancel()
	}
	c.generation++
	ctx, cancel := context.WithCancel(c.ctx)
	op := &operation{generation: c.generation, ctx: ctx, cancel: cancel, done: make(chan struct{})}
	c.tail = op
	c.state = Starting
	if stop {
		c.state = Stopping
	}
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer close(op.done)
		defer func() {
			c.mu.Lock()
			defer c.mu.Unlock()
			if c.tail == op && !c.closed && ctx.Err() != nil && c.state != Failed {
				c.state = Stopped
			}
		}()
		defer cancel()
		// Serialize resource destruction before a replacement session binds the same port.
		if previous != nil {
			<-previous.done
		}
		if ctx.Err() != nil {
			return
		}
		if stop {
			c.publish(op, Event{State: Stopped})
			return
		}
		c.run(op, config)
	}()
	return c.generation
}
func (c *Controller) Close() {
	c.once.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.state = Closed
		c.cancel()
		c.mu.Unlock()
		c.wg.Wait()
		close(c.events)
	})
}
func (c *Controller) publish(op *operation, e Event) bool {
	c.mu.Lock()
	current := !c.closed && c.tail == op && op.ctx.Err() == nil
	if current && e.State != "" {
		c.state = e.State
	}
	c.mu.Unlock()
	if !current {
		return false
	}
	e.Generation = op.generation
	select {
	case c.events <- e:
		return true
	case <-op.ctx.Done():
		return false
	}
}
func (c *Controller) Send(generation uint64, cmd peer.Command) error {
	return c.Prepare(generation, cmd)()
}

// Prepare captures both the session generation and transport connection before
// presentation schedules the asynchronous write.
func (c *Controller) Prepare(generation uint64, cmd peer.Command) func() error {
	c.mu.Lock()
	op := c.tail
	if c.closed || op == nil || op.generation != generation || op.ctx.Err() != nil || (op.client == nil && op.local == nil) {
		c.mu.Unlock()
		return func() error { return errors.New("session changed or is not connected") }
	}
	var execute func() error
	if op.local != nil {
		cmd.Fleet = append([]game.Placement(nil), cmd.Fleet...)
		execute = func() error {
			request := localCommand{command: cmd, result: make(chan error, 1)}
			select {
			case op.local <- request:
			case <-op.ctx.Done():
				return errors.New("session ended")
			}
			select {
			case err := <-request.result:
				return err
			case <-op.ctx.Done():
				return errors.New("session ended")
			}
		}
	} else {
		execute = op.client.Prepare(cmd)
	}
	c.mu.Unlock()
	return func() error {
		c.mu.Lock()
		valid := !c.closed && c.tail == op && op.ctx.Err() == nil
		c.mu.Unlock()
		if !valid {
			return errors.New("session changed or is not connected")
		}
		return execute()
	}
}
func (c *Controller) UpdateInvitation(generation uint64, code string) error {
	i, err := peer.ParseInvitation(code)
	if err != nil {
		return err
	}
	c.mu.Lock()
	op := c.tail
	if c.closed || op == nil || op.generation != generation || op.client == nil || op.ctx.Err() != nil {
		c.mu.Unlock()
		return errors.New("session is no longer active")
	}
	client := op.client
	c.mu.Unlock()
	return client.UpdateInvitation(i)
}
func (c *Controller) run(op *operation, config Config) {
	fail := func(err error) { c.publish(op, Event{State: Failed, Error: err.Error()}) }
	c.publish(op, Event{State: Starting})
	if config.Solo {
		if config.Host || config.Join != "" || config.Human < 0 || config.Human > 1 {
			fail(errors.New("choose a valid solo faction without host or join"))
			return
		}
		c.runLocal(op, config.Human)
		return
	}
	if config.Port < 0 || config.Port > 65535 {
		fail(errors.New("port must be 0-65535"))
		return
	}
	if config.Advertise != "" {
		if err := peer.ValidAddress(config.Advertise); err != nil {
			fail(err)
			return
		}
	}
	if config.Host && config.Join != "" {
		fail(errors.New("choose host or join"))
		return
	}
	var server *peer.Server
	var invite peer.Invitation
	var err error
	setup := Event{State: Connecting, Host: config.Host}
	if config.Host {
		server, err = c.deps.host(net.JoinHostPort("", strconv.Itoa(config.Port)))
		if err != nil {
			fail(err)
			return
		}
		defer server.Close()
		setup.Port = server.Port()
		public, e := server.UpdateAddresses(routes(c.deps.addresses(server.Port()), config.Advertise, ""))
		if e != nil {
			fail(e)
			return
		}
		setup.Invitation, err = public.Encode()
		if err != nil {
			fail(err)
			return
		}
		invite = server.LocalInvite()
	} else {
		invite, err = peer.ParseInvitation(config.Join)
		if err != nil {
			fail(err)
			return
		}
	}
	client, err := c.deps.connect(op.ctx, invite)
	if err != nil {
		fail(err)
		return
	}
	defer client.Close()
	c.mu.Lock()
	op.client = client
	c.mu.Unlock()
	defer func() { c.mu.Lock(); op.client = nil; c.mu.Unlock() }()
	if !c.publish(op, setup) {
		return
	}
	// Router discovery is owned and joined here, including late completion after
	// cancellation. No UI command can create a lease that outlives Close.
	mappingCtx, mappingCancel := context.WithCancel(op.ctx)
	var mappingWG sync.WaitGroup
	defer func() { mappingCancel(); mappingWG.Wait() }()
	if server != nil {
		mappingWG.Add(1)
		go func() { defer mappingWG.Done(); c.manageRoutes(op, mappingCtx, server, config) }()
	}

	hadState := false
	for {
		select {
		case <-op.ctx.Done():
			return
		case m, ok := <-client.Updates:
			if !ok {
				if op.ctx.Err() == nil {
					fail(errors.New("connection closed"))
				}
				return
			}
			e := Event{Message: m}
			switch m.Type {
			case "state":
				e.State = Active
				hadState = true
			case "connecting", "offline":
				e.State = Connecting
				if hadState {
					e.State = Reconnecting
				}
			case "fatal":
				e.State = Failed
				e.Error = m.Error
			}
			if !c.publish(op, e) {
				return
			}
			if m.Type == "fatal" {
				return
			}
		}
	}
}
func routes(local []string, manual, mapped string) []string {
	seen := map[string]bool{}
	result := []string{}
	candidates := append([]string{mapped, manual}, local...)
	for _, a := range candidates {
		if a != "" && !seen[a] {
			seen[a] = true
			result = append(result, a)
		}
	}
	return result
}
func (e Event) String() string { return fmt.Sprintf("session %d: %s", e.Generation, e.State) }

// Route discovery and invitation publication are serialized here. Refreshing
// local interfaces also works with --no-map and while a router is unavailable.
func (c *Controller) manageRoutes(op *operation, ctx context.Context, server *peer.Server, config Config) {
	updates := make(chan netsetup.Update, 8)
	var wg sync.WaitGroup
	if !config.NoMap {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.deps.maintain(ctx, server.Port(), func(u netsetup.Update) {
				select {
				case updates <- u:
				case <-ctx.Done():
				}
			})
		}()
	} else {
		u := netsetup.Update{State: "unavailable", Notice: "Router mapping disabled. LAN/IPv6/manual routes only."}
		c.publish(op, Event{Router: &u})
	}
	defer wg.Wait()
	ticker := time.NewTicker(c.deps.refresh)
	defer ticker.Stop()
	mapped, last := "", ""
	var expires time.Time
	publish := func() {
		if !expires.IsZero() && !time.Now().Before(expires) {
			mapped = ""
		}
		current := strings.Join(routes(c.deps.addresses(server.Port()), config.Advertise, mapped), "\n")
		if current == last {
			return
		}
		invite, err := server.UpdateAddresses(strings.Split(current, "\n"))
		if err != nil {
			return
		}
		code, err := invite.Encode()
		if err != nil {
			return
		}
		last = current
		c.publish(op, Event{Invitation: code})
	}
	publish()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			publish()
		case u := <-updates:
			if u.State == "active" && !u.Expires.IsZero() && !time.Now().Before(u.Expires) {
				u.State = "unavailable"
				u.Address = ""
				u.Notice = "Router lease expired; waiting for a fresh route."
			}
			if u.State == "active" {
				expires = u.Expires
				mapped = u.Address
			}
			if u.State == "unavailable" {
				mapped = ""
			}
			publish()
			c.publish(op, Event{Router: &u})
		}
	}
}
