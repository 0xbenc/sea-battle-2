package peer

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

func publicConn(conn net.Conn) bool {
	if tc, ok := conn.(*tls.Conn); ok {
		_, public := tc.NetConn().(*websocket.Conn)
		return public
	}
	return false
}
func transport(conn net.Conn) string {
	if publicConn(conn) {
		return "public"
	}
	return "direct"
}
func scanner(conn net.Conn) *bufio.Scanner {
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 4096), maxMessage)
	return sc
}
func scanMessage(sc *bufio.Scanner) (Message, error) {
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return Message{}, err
		}
		return Message{}, errors.New("host closed connection")
	}
	return decodeMessage(sc.Bytes())
}

// probeDirect proves a second authenticated path without claiming or disturbing
// the occupied seat. Only this client's resumption key can authorize the probe.
func (c *Client) probeDirect(ctx context.Context, addr string) (net.Conn, *bufio.Scanner, error) {
	i := c.Invitation()
	conn, err := c.dialTransport(ctx, addr, i.Fingerprint)
	if err != nil {
		return nil, nil, err
	}
	stop := context.AfterFunc(ctx, func() { abortConn(conn) })
	defer stop()
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}
	err = json.NewEncoder(conn).Encode(Command{Type: "probe", Version: Version, Token: i.Token, Resume: c.resume})
	sc := scanner(conn)
	if err == nil {
		var m Message
		m, err = scanMessage(sc)
		if err == nil && m.Type != "probe_ok" {
			err = errors.New("host did not accept route probe")
		}
	}
	if err != nil || ctx.Err() != nil {
		abortConn(conn)
		if err == nil {
			err = ctx.Err()
		}
		return nil, nil, err
	}
	return conn, sc, nil
}

// negotiate runs before exposing the first public game state to the UI. Gameplay
// commands therefore cannot straddle the handoff. Failed probes retain the public
// socket; activation uses the same seat, and only a state acknowledgement allows
// the host to retire its tunnel.
func (c *Client) negotiate(old net.Conn) (net.Conn, *bufio.Scanner, Message, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 4*time.Second)
	defer cancel()
	type result struct {
		conn net.Conn
		sc   *bufio.Scanner
		err  error
	}
	addresses := c.Invitation().Addresses
	if c.route != "" {
		addresses = append([]string{c.route}, addresses...)
	}
	results := make(chan result)
	count := 0
	// Every worker is joined, and all losing sockets are closed.
	var workers int
	for _, addr := range addresses {
		if !strings.HasPrefix(addr, "wss://") {
			workers++
		}
	}
	if workers == 0 {
		return nil, nil, Message{}, nil
	}
	finished := make(chan struct{}, workers)
	defer func() {
		cancel()
		for n := 0; n < workers; n++ {
			<-finished
		}
	}()
	for _, addr := range addresses {
		if strings.HasPrefix(addr, "wss://") {
			continue
		}
		count++
		go func(addr string) {
			defer func() { finished <- struct{}{} }()
			conn, sc, err := c.probeDirect(ctx, addr)
			select {
			case results <- result{conn, sc, err}:
			case <-ctx.Done():
				if conn != nil {
					abortConn(conn)
				}
			}
		}(addr)
	}
	var candidate result
	for n := 0; n < count; n++ {
		select {
		case r := <-results:
			if r.err == nil {
				candidate = r
				goto found
			}
		case <-ctx.Done():
			return nil, nil, Message{}, nil
		}
	}
	return nil, nil, Message{}, nil
found:
	// Cancel losing probes, but give the selected connection its own activation
	// deadline; its probe cancellation callback has already been detached.
	cancel()
	conn := candidate.conn
	activation, stop := context.WithTimeout(c.ctx, 5*time.Second)
	defer stop()
	closeOnCancel := context.AfterFunc(activation, func() { abortConn(conn) })
	defer closeOnCancel()
	deadline, _ := activation.Deadline()
	conn.SetDeadline(deadline)
	c.writeMu.Lock()
	c.mu.Lock()
	c.conn = conn
	c.ready = false
	c.mu.Unlock()
	c.writeMu.Unlock()
	err := c.Send(Command{Type: "hello", Version: Version, Token: c.Invitation().Token, Resume: c.resume, Negotiate: true})
	var m Message
	for err == nil {
		m, err = scanMessage(candidate.sc)
		if err != nil {
			break
		}
		if m.Type == "invitation" {
			err = c.UpdateInvitation(*m.Invitation)
			continue
		}
		if m.Type == "state" {
			break
		}
		err = errors.New("direct route activation was rejected")
	}
	if err != nil {
		abortConn(conn)
		c.mu.Lock()
		c.conn = old
		c.mu.Unlock()
		// Keep the existing public stream if activation failed before replacing its
		// seat. If the host already replaced it, normal reconnect recovers over the
		// still-running public connector; no direct confirmation was sent.
		if c.ctx.Err() != nil {
			return nil, nil, Message{}, c.ctx.Err()
		}
		c.emit(Message{Type: "route_failed", Error: "Direct activation failed; retaining public route: " + err.Error()})
		return nil, nil, Message{}, nil
	}
	conn.SetDeadline(time.Time{})
	abortConn(old)
	return conn, candidate.sc, m, nil
}
