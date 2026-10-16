package peer

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sea-battle-2/internal/game"
	"slices"
	"sync"
	"time"
)

// Client reconnects to the same host identity. Commands are never replayed;
// the next authoritative state resolves whether an interrupted shot happened.
type Client struct {
	Updates chan Message
	invite  Invitation
	resume  string
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	writeMu sync.Mutex
	ready   bool
	wake    chan struct{}
	conn    net.Conn
	done    chan struct{}
}

func Connect(ctx context.Context, i Invitation) (*Client, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	resume, err := secret()
	if err != nil {
		return nil, err
	}
	i.Addresses = append([]string(nil), i.Addresses...)
	ctx, cancel := context.WithCancel(ctx)
	c := &Client{Updates: make(chan Message, 32), invite: i, resume: resume, ctx: ctx, cancel: cancel, done: make(chan struct{}), wake: make(chan struct{}, 1)}
	go c.run()
	return c, nil
}
func (c *Client) Close() {
	c.cancel()
	c.mu.Lock()
	if c.conn != nil {
		abortConn(c.conn)
	}
	c.mu.Unlock()
	<-c.done
}
func (c *Client) emit(m Message) bool {
	select {
	case c.Updates <- m:
		return true
	case <-c.ctx.Done():
		return false
	}
}

// Prepare binds an intent to the current connection. A queued UI command
// cannot silently move to a replacement connection after a disconnect.
func (c *Client) Prepare(cmd Command) func() error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return func() error { return errors.New("connection is unavailable") }
	}
	cmd.Fleet = append([]game.Placement(nil), cmd.Fleet...)
	return func() error { return c.sendOn(conn, cmd) }
}
func (c *Client) Send(cmd Command) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	return c.sendOn(conn, cmd)
}
func (c *Client) sendOn(conn net.Conn, cmd Command) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.mu.Lock()
	valid := conn != nil && conn == c.conn && c.ctx.Err() == nil && (c.ready || cmd.Type == "hello" || cmd.Type == "ping")
	c.mu.Unlock()
	if !valid {
		return errors.New("connection changed or is unavailable; wait for the current board")
	}
	conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	if err := json.NewEncoder(conn).Encode(cmd); err != nil {
		abortConn(conn)
		return errors.New("connection interrupted; waiting for host state")
	}
	return nil
}
func (c *Client) dial() (net.Conn, error) {
	invite := c.Invitation()
	ctx, cancel := context.WithTimeout(c.ctx, 6*time.Second)
	defer cancel()
	type result struct {
		conn net.Conn
		err  error
	}
	results := make(chan result)
	var wg sync.WaitGroup
	for _, addr := range invite.Addresses {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			d := tls.Dialer{NetDialer: &net.Dialer{Timeout: 5 * time.Second}, Config: pinnedConfig(invite.Fingerprint)}
			conn, err := d.DialContext(ctx, "tcp", addr)
			select {
			case results <- result{conn, err}:
			case <-ctx.Done():
				if conn != nil {
					abortConn(conn)
				}
			}
		}(addr)
	}
	// Wait for all dialers on every exit: no orphaned successful connections.
	defer wg.Wait()
	for range invite.Addresses {
		select {
		case r := <-results:
			if r.err == nil {
				cancel()
				return r.conn, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, errors.New("no invitation address accepted the pinned TLS connection")
}

var errRejected = errors.New("host rejected invitation")

func (c *Client) run() {
	defer close(c.done)
	defer close(c.Updates)
	delay := time.Second
	for c.ctx.Err() == nil {
		if err := c.Invitation().Validate(); err != nil {
			c.emit(Message{Type: "fatal", Error: err.Error()})
			return
		}
		c.emit(Message{Type: "connecting"})
		conn, err := c.dial()
		if err == nil {
			c.mu.Lock()
			c.conn = conn
			c.ready = false
			c.mu.Unlock()
			err = c.Send(Command{Type: "hello", Version: Version, Token: c.Invitation().Token, Resume: c.resume})
			if err == nil {
				err = c.read(conn)
			}
			c.mu.Lock()
			if c.ready {
				delay = time.Second
			}
			c.ready = false
			c.conn = nil
			abortConn(conn)
			c.mu.Unlock()
		}
		if c.ctx.Err() != nil || errors.Is(err, errRejected) {
			return
		}
		if !c.emit(Message{Type: "offline", Error: fmt.Sprintf("Disconnected; retrying in %s. %v", delay, err)}) {
			return
		}
		select {
		case <-time.After(delay):
		case <-c.wake:
		case <-c.ctx.Done():
			return
		}
		delay = min(5*time.Second, delay*2)
	}
}
func (c *Client) read(conn net.Conn) error {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if c.Send(Command{Type: "ping"}) != nil {
					return
				}
			case <-stop:
				return
			case <-c.ctx.Done():
				abortConn(conn)
				return
			}
		}
	}()
	defer func() { close(stop); <-done }()
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 4096), maxMessage)
	for {
		conn.SetReadDeadline(time.Now().Add(35 * time.Second))
		if !sc.Scan() {
			break
		}
		m, err := decodeMessage(sc.Bytes())
		if err != nil {
			return err
		}
		if m.Type == "state" {
			c.mu.Lock()
			c.ready = true
			c.mu.Unlock()
			select {
			case <-c.wake:
			default:
			}
		}
		if m.Type == "invitation" {
			if err := c.UpdateInvitation(*m.Invitation); err != nil {
				return err
			}
		}
		if m.Type == "fatal" {
			c.emit(m)
			return errRejected
		}
		if m.Type != "pong" && !c.emit(m) {
			return c.ctx.Err()
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return errors.New("host closed connection")
}

// Validate the bounded wire shape before presentation indexes into faction art.
func validView(v *game.View) bool {
	if v == nil || v.Player < 0 || v.Player > 1 || v.Turn < 0 || v.Turn > 1 || v.Winner < -1 || v.Winner > 1 || v.Match < 1 {
		return false
	}
	if v.Phase != "placement" && v.Phase != "battle" && v.Phase != "finished" {
		return false
	}
	if len(v.Own) != 0 && game.ValidateFleet(v.Own) != nil {
		return false
	}
	if v.Ready[v.Player] && len(v.Own) != 5 {
		return false
	}
	if len(v.Events) > 200 {
		return false
	}
	for i, e := range v.Events {
		if e.Seq != i+1 || e.Player < 0 || e.Player > 1 || !e.At.Valid() || e.Ship < -1 || e.Ship > 4 || e.Sunk && (!e.Hit || e.Ship < 0) {
			return false
		}
	}
	for i := range v.Incoming {
		if v.Incoming[i] > game.Sunk || v.Outgoing[i] > game.Sunk {
			return false
		}
	}
	return true
}

// Invitation returns a copy of the current reconnect endpoints and fixed identity.
func (c *Client) Invitation() Invitation {
	c.mu.Lock()
	defer c.mu.Unlock()
	i := c.invite
	i.Addresses = append([]string(nil), i.Addresses...)
	return i
}

// UpdateInvitation can update routes, never the authenticated host or seat.
// It also allows a new pasted invitation to recover a completely broken path.
func (c *Client) UpdateInvitation(i Invitation) error {
	if err := i.Validate(); err != nil {
		return err
	}
	c.mu.Lock()
	if i.Fingerprint != c.invite.Fingerprint || i.Token != c.invite.Token || i.Expires != c.invite.Expires || i.Version != c.invite.Version {
		c.mu.Unlock()
		return errors.New("updated invitation belongs to a different session")
	}
	wake := !c.ready && !slices.Equal(i.Addresses, c.invite.Addresses)
	i.Addresses = append([]string(nil), i.Addresses...)
	c.invite = i
	c.mu.Unlock()
	if wake {
		select {
		case c.wake <- struct{}{}:
		default:
		}
	}
	return nil
}
