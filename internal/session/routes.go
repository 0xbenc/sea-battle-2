package session

import (
	"context"
	"strings"
	"sync"
	"time"

	"sea-battle-2/internal/netsetup"
	"sea-battle-2/internal/peer"
	"sea-battle-2/internal/tunnel"
)

// One owner serializes every advertised route and all connector lifetimes.
func (c *Controller) manageRoutes(op *operation, ctx context.Context, server *peer.Server, config Config) {
	updates := make(chan netsetup.Update, 8)
	var mappingWG sync.WaitGroup
	if !config.NoMap {
		mappingWG.Add(1)
		go func() {
			defer mappingWG.Done()
			c.deps.maintain(ctx, server.Port(), func(u netsetup.Update) {
				select {
				case updates <- u:
				case <-ctx.Done():
				}
			})
		}()
	}
	defer mappingWG.Wait()
	type result struct {
		handle tunnel.Handle
		err    error
	}
	results := make(chan result)
	var launching context.CancelFunc
	var launchDone chan struct{}
	var active tunnel.Handle
	var connectorDone <-chan struct{}
	var retry <-chan time.Time
	var retryTimer *time.Timer
	defer func() {
		if retryTimer != nil {
			retryTimer.Stop()
		}
		if launching != nil {
			launching()
			<-launchDone
		}
		if active != nil {
			active.Close()
		}
	}()
	mapped, public, last := "", "", ""
	ready := !config.AutoPublic
	var expires time.Time
	network := func(state, notice, detail string) {
		c.publish(op, Event{Network: state, Notice: notice, Detail: detail})
	}
	publish := func() {
		if !ready {
			return
		}
		if !expires.IsZero() && !time.Now().Before(expires) {
			mapped = ""
		}
		addresses := routes(c.localAddresses(config, server.Port()), config.Advertise, mapped)
		if public != "" {
			addresses = append([]string{public}, addresses...)
		}
		current := strings.Join(addresses, "\n")
		if current == last {
			return
		}
		invite, err := server.UpdateAddresses(addresses)
		if err != nil {
			return
		}
		code, err := invite.Encode()
		if err != nil {
			return
		}
		last = current
		c.publish(op, Event{Invitation: code, Port: server.Port()})
	}
	schedule := func(d time.Duration) {
		if retryTimer != nil {
			retryTimer.Stop()
		}
		retryTimer = time.NewTimer(d)
		retry = retryTimer.C
	}
	begin := func() {
		if !config.AutoPublic || launching != nil || active != nil {
			return
		}
		launchCtx, cancel := context.WithCancel(ctx)
		launching = cancel
		launchDone = make(chan struct{})
		done := launchDone
		network("preparing", "Preparing a secure invitation…", "")
		go func() {
			defer close(done)
			h, err := c.deps.tunnel(launchCtx, config.tunnelPort, func(notice string) { network("preparing", notice, "") })
			select {
			case results <- result{h, err}:
			case <-launchCtx.Done():
				if h != nil {
					h.Close()
				}
			}
		}()
	}
	if config.AutoPublic {
		begin()
	} else {
		publish()
	}
	ticker := time.NewTicker(c.deps.refresh)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-results:
			<-launchDone
			// The handle owns cancellation of the process. Retain the launch context's
			// cancellation until that handle closes, without cancelling a live connector.
			cancel := launching
			launching = nil
			ready = true
			if r.err != nil {
				cancel()
				network("unavailable", "Public setup unavailable. Direct connections are still available.", r.err.Error())
				schedule(30 * time.Second)
			} else {
				active = &ownedTunnel{Handle: r.handle, cancel: cancel}
				connectorDone = active.Done()
				public = active.Address()
				mode, settled := server.GuestLink()
				if mode == "direct" && settled {
					public = ""
					active.Close()
					active = nil
					connectorDone = nil
					network("direct", "Connected directly. Public connection is off.", "")
				} else {
					network("ready", "Invitation ready. Share it privately with your friend.", "")
				}
			}
			publish()
		case <-server.LinkChanges():
			mode, settled := server.GuestLink()
			if settled && mode == "direct" {
				if retryTimer != nil {
					retryTimer.Stop()
				}
				retry = nil
				if launching != nil {
					launching()
					<-launchDone
					launching = nil
				}
				ready = true
				public = ""
				publish()
				if active != nil {
					active.Close()
					active = nil
					connectorDone = nil
				}
				notice := "Connected directly."
				if config.AutoPublic {
					notice = "Connected directly. Public connection is off."
				}
				network("direct", notice, "")
			} else if settled && mode == "public" {
				network("public", "Connected securely through the public connection.", "")
			} else if mode == "" && config.AutoPublic && active == nil && launching == nil {
				schedule(3 * time.Second)
			}
		case <-connectorDone:
			active.Close()
			active = nil
			connectorDone = nil
			public = ""
			publish()
			network("recovering", "Public connection interrupted. Preparing a new invitation…", "")
			schedule(time.Second)
		case <-retry:
			retry = nil
			mode, settled := server.GuestLink()
			if mode != "direct" || !settled {
				begin()
			}
		case <-ticker.C:
			publish()
		case u := <-updates:
			if u.State == "active" && !u.Expires.IsZero() && !time.Now().Before(u.Expires) {
				u.State = "unavailable"
				u.Address = ""
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

type ownedTunnel struct {
	tunnel.Handle
	cancel context.CancelFunc
}

func (t *ownedTunnel) Close() { t.cancel(); t.Handle.Close() }
