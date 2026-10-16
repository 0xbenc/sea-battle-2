package netsetup

import (
	"context"
	"time"
)

// Update describes a router lease, never an assertion of Internet reachability.
type Update struct {
	State                     string // searching, active, retrying, unavailable
	Address, Protocol, Notice string
	Expires                   time.Time
}

func wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(max(0, d))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return ctx.Err() == nil
	}
}

// Maintain owns discovery, lease renewal, recovery, and best-effort deletion.
// Discovery continues after a router comes back or the network changes.
func Maintain(ctx context.Context, port int, report func(Update)) {
	maintain(ctx, func(ctx context.Context) (*Mapping, error) { return Discover(ctx, port) }, report)
}
func maintain(ctx context.Context, discover func(context.Context) (*Mapping, error), report func(Update)) {
	retry := 5 * time.Second
	for ctx.Err() == nil {
		report(Update{State: "searching", Notice: "Looking for a temporary mapping on the local router."})
		attempt, cancel := context.WithTimeout(ctx, 8*time.Second)
		m, err := discover(attempt)
		cancel()
		if err == nil {
			retry = 5 * time.Second
			// Even a mapping returned concurrently with cancellation must be released.
			m.run(ctx, report)
		} else if ctx.Err() == nil {
			report(Update{State: "unavailable", Notice: "No router mapping available. Retrying; LAN/IPv6/manual routes remain available."})
		}
		if !wait(ctx, retry) {
			return
		}
		retry = min(30*time.Second, retry*2)
	}
}
func (m *Mapping) run(ctx context.Context, report func(Update)) {
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		m.renew(cleanup, 0)
	}()
	if ctx.Err() != nil {
		return
	}
	expires := m.expires
	if expires.IsZero() {
		expires = time.Now().Add(m.duration)
	}
	address := m.Address
	if !time.Now().Before(expires) {
		report(Update{State: "unavailable", Protocol: m.Protocol, Notice: "Router lease expired; rediscovering a route."})
		return
	}
	report(Update{State: "active", Address: address, Protocol: m.Protocol, Expires: expires})
	next := time.Until(expires) / 2
	retry := time.Second
	for wait(ctx, next) {
		if !time.Now().Before(expires) {
			report(Update{State: "unavailable", Protocol: m.Protocol, Notice: "Router lease expired; rediscovering a route."})
			return
		}
		started := time.Now()
		attempt, cancel := context.WithDeadline(ctx, minTime(expires, started.Add(4*time.Second)))
		addr, d, err := m.renew(attempt, leaseSeconds)
		cancel()
		if ctx.Err() != nil {
			return
		}
		if err == nil && d > 0 && started.Add(d).After(time.Now()) {
			address = addr
			expires = started.Add(d)
			retry = time.Second
			next = time.Until(expires) / 2
			report(Update{State: "active", Address: addr, Protocol: m.Protocol, Expires: expires})
			continue
		}
		if !time.Now().Before(expires) {
			report(Update{State: "unavailable", Protocol: m.Protocol, Notice: "Router lease expired; rediscovering a route."})
			return
		}
		// Keep the last confirmed mapping until its lease expires. A lost reply
		// does not justify deleting a route that may still carry a live battle.
		report(Update{State: "retrying", Address: address, Protocol: m.Protocol, Expires: expires, Notice: "Router renewal delayed; retrying before the confirmed lease expires."})
		next = min(retry, time.Until(expires))
		retry = min(10*time.Second, retry*2)
	}
}
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
