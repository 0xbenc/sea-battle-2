// netcheck exercises the real multiplayer session without a terminal UI.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sea-battle-2/internal/game"
	"sea-battle-2/internal/peer"
	"sea-battle-2/internal/session"
)

func run() error {
	host := flag.Bool("host", false, "host a test match")
	file := flag.String("invitation", "", "private invitation file to write (host) or read (guest)")
	port := flag.Int("port", 0, "host port")
	advertise := flag.String("advertise", "", "additional host:port route")
	websocketPort := flag.Int("websocket-port", 0, "loopback tunnel port")
	only := flag.String("only-route", "", "test only this invitation route")
	mapping := flag.Bool("map", false, "test temporary router mappings")
	via := flag.String("connect-address", "", "additional guest route")
	timeout := flag.Duration("timeout", 90*time.Second, "overall deadline")
	flag.Parse()
	if *file == "" {
		return fmt.Errorf("--invitation is required")
	}
	cfg := session.Config{Host: *host, Port: *port, Advertise: *advertise, NoMap: !*mapping, JoinAddress: *via, WebSocketPort: *websocketPort}
	if !*host {
		b, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		cfg.Join = string(b)
		if *only != "" {
			i, err := peer.ParseInvitation(cfg.Join)
			if err != nil {
				return err
			}
			i.Addresses = []string{*only}
			cfg.Join, err = i.Encode()
			if err != nil {
				return err
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	c := session.New(ctx)
	defer c.Close()
	gen := c.Start(cfg)
	ready := false
	lastShot := -1
	target := 0
	started := time.Now()
	invitationWritten := false
	for {
		e, ok := c.Next()
		if !ok {
			return fmt.Errorf("test ended before completion: %v", ctx.Err())
		}
		if e.Invitation != "" && *host {
			if invitationWritten {
				if err := refreshInvitation(*file, e.Invitation); err != nil {
					return err
				}
				continue
			}
			// Never print bearer credentials. Exclusive creation avoids overwriting files.
			f, err := os.OpenFile(*file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				return err
			}
			_, err = f.WriteString(e.Invitation + "\n")
			closeErr := f.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
			i, _ := peer.ParseInvitation(e.Invitation)
			invitationWritten = true
			fmt.Printf("LISTEN port=%d routes=%v\n", e.Port, i.Addresses)
		}
		if e.Router != nil {
			fmt.Printf("ROUTER state=%s protocol=%s address=%s notice=%s\n", e.Router.State, e.Router.Protocol, e.Router.Address, e.Router.Notice)
		}
		if e.State == session.Failed {
			return fmt.Errorf("session failed: %s", e.Error)
		}
		if e.Message.Type == "offline" {
			fmt.Println(e.Message.Error)
		}
		if e.Message.Type == "error" || e.Message.Type == "fatal" {
			return fmt.Errorf("protocol: %s", e.Message.Error)
		}
		v := e.Message.View
		if v == nil {
			continue
		}
		if !ready {
			fmt.Printf("AUTHENTICATED player=%d route=%s elapsed=%s\n", v.Player, e.Message.Route, time.Since(started).Round(time.Millisecond))
			if err := c.Send(gen, peer.Command{Type: "ready", Match: v.Match, Fleet: game.RandomFleet(uint64(v.Player + 11))}); err != nil {
				return err
			}
			ready = true
		}
		if v.Phase == "finished" {
			fmt.Printf("PASS match=%d shots=%d winner=%d elapsed=%s\n", v.Match, len(v.Events), v.Winner, time.Since(started).Round(time.Millisecond))
			// Allow the remote peer to receive the final authoritative state before shutdown.
			if *host {
				time.Sleep(time.Second)
			}
			return nil
		}
		if v.Phase == "battle" && v.Connected[0] && v.Connected[1] && v.Turn == v.Player && lastShot != len(v.Events) {
			if target >= 100 {
				return fmt.Errorf("exhausted targets")
			}
			err := c.Send(gen, peer.Command{Type: "fire", Match: v.Match, Expected: len(v.Events), At: game.Coord{X: target % 10, Y: target / 10}})
			if err != nil {
				return err
			}
			lastShot = len(v.Events)
			target++
		}
	}
}

// Replace only the invitation this process created, using a private atomic write.
func refreshInvitation(path, code string) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".netcheck-invite-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.WriteString(code + "\n")
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
