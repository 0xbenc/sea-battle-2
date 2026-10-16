package session

import (
	"errors"
	"math/rand/v2"
	"sea-battle-2/internal/bot"
	"sea-battle-2/internal/game"
	"sea-battle-2/internal/peer"
	"time"
)

type localCommand struct {
	command peer.Command
	result  chan error
}

// One owner serializes human commands and bot turns; cancellation owns the timer.
func (c *Controller) runLocal(op *operation, human int, difficulty bot.Difficulty) {
	commands := make(chan localCommand)
	c.mu.Lock()
	op.local = commands
	c.mu.Unlock()
	g := game.New(human)
	opponent := 1 - human
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	deploy := func() { _ = g.SetFleet(opponent, game.RandomFleet(rand.Uint64())) }
	deploy()
	if !c.publish(op, Event{State: Connecting, Solo: true}) {
		return
	}
	publish := func() bool {
		v := g.View(human)
		v.Connected = [2]bool{true, true}
		return c.publish(op, Event{State: Active, Message: peer.Message{Type: "state", View: &v}})
	}
	if !publish() {
		return
	}
	var timer *time.Timer
	var turn <-chan time.Time
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	schedule := func() {
		if g.Phase == "battle" && g.Turn == opponent && turn == nil {
			timer = time.NewTimer(c.deps.botDelay)
			turn = timer.C
		}
	}
	for {
		schedule()
		select {
		case <-op.ctx.Done():
			return
		case <-turn:
			turn = nil
			if op.ctx.Err() != nil {
				return
			}
			at, ok := bot.Decide(op.ctx, difficulty, bot.Observe(g.View(opponent)), rng)
			if op.ctx.Err() != nil {
				return
			}
			if !ok {
				c.publish(op, Event{State: Failed, Error: "Bot has no legal target"})
				return
			}
			if err := g.Fire(opponent, at, len(g.Events)); err != nil {
				c.publish(op, Event{State: Failed, Error: err.Error()})
				return
			}
			if !publish() {
				return
			}
		case request := <-commands:
			if op.ctx.Err() != nil {
				return
			}
			cmd := request.command
			var err error
			if cmd.Match != g.Match {
				err = errors.New("match changed; choose again")
			} else {
				switch cmd.Type {
				case "ready":
					err = g.SetFleet(human, cmd.Fleet)
				case "fire":
					err = g.Fire(human, cmd.At, cmd.Expected)
				case "rematch":
					err = g.RequestRematch(human)
					if err == nil {
						err = g.RequestRematch(opponent)
						if err == nil {
							deploy()
						}
					}
				default:
					err = errors.New("unsupported local command")
				}
			}
			request.result <- err
			if err == nil && !publish() {
				return
			}
		}
	}
}
