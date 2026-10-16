package session

import (
	"context"
	"math/rand/v2"
	"sea-battle-2/internal/bot"
	"sea-battle-2/internal/game"
	"sea-battle-2/internal/peer"
	"testing"
	"time"
)

func localView(t *testing.T, c *Controller) game.View {
	t.Helper()
	e := await(t, c, func(e Event) bool { return e.Message.View != nil || e.State == Failed })
	if e.State == Failed {
		t.Fatal(e.Error)
	}
	return *e.Message.View
}
func TestSoloFullMatchesBothFactionsAndRematches(t *testing.T) {
	for human := 0; human < 2; human++ {
		c := New(context.Background())
		c.deps.botDelay = 0
		c.deps.host = func(string) (*peer.Server, error) { t.Fatal("solo opened network host"); return nil, nil }
		gen := c.Start(Config{Solo: true, Human: human})
		rng := rand.New(rand.NewPCG(21, 33))
		for match := 1; match <= 12; match++ {
			v := localView(t, c)
			if v.Player != human || v.Match != match || !v.Ready[1-human] || v.Ready[human] || v.Turn != (human+match-1)%2 {
				t.Fatalf("bad deployment: %+v", v)
			}
			if err := c.Send(gen, peer.Command{Type: "ready", Match: match, Fleet: game.RandomFleet(uint64(match))}); err != nil {
				t.Fatal(err)
			}
			v = localView(t, c)
			for v.Phase == "battle" {
				if v.Turn == human {
					at, ok := bot.Choose(bot.Observe(v), rng)
					if !ok {
						t.Fatal("no human target")
					}
					cmd := peer.Command{Type: "fire", Match: match, Expected: len(v.Events), At: at}
					if err := c.Send(gen, cmd); err != nil {
						t.Fatal(err)
					}
					if err := c.Send(gen, cmd); err == nil {
						t.Fatal("repeated shot accepted")
					}
				}
				v = localView(t, c)
			}
			if v.Phase != "finished" || v.Winner < 0 || len(v.Events) > 199 {
				t.Fatal("bad result")
			}
			if err := c.Send(gen, peer.Command{Type: "rematch", Match: match}); err != nil {
				t.Fatal(err)
			}
			if err := c.Send(gen, peer.Command{Type: "rematch", Match: match}); err == nil {
				t.Fatal("stale match accepted")
			}
		}
		c.Close()
	}
}
func TestSoloCancellationAndPreparedCommand(t *testing.T) {
	c := New(context.Background())
	defer c.Close()
	gen := c.Start(Config{Solo: true, Human: 1})
	localView(t, c)
	ready := peer.Command{Type: "ready", Match: 1, Fleet: game.RandomFleet(5)}
	prepared := c.Prepare(gen, ready)
	if err := prepared(); err != nil {
		t.Fatal(err)
	}
	v := localView(t, c)
	if err := c.Send(gen, peer.Command{Type: "fire", Match: 1, Expected: len(v.Events), At: game.Coord{}}); err != nil {
		t.Fatal(err)
	}
	localView(t, c)
	start := time.Now()
	c.Stop()
	await(t, c, func(e Event) bool { return e.State == Stopped })
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("stop waited on bot timer")
	}
	c.Start(Config{Solo: true, Human: 0})
	v = localView(t, c)
	if v.Player != 0 || len(v.Events) != 0 {
		t.Fatal("old bot leaked into new session")
	}
	if err := prepared(); err == nil {
		t.Fatal("prepared command crossed generations")
	}
}
