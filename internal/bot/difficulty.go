package bot

import (
	"context"
	"math/rand/v2"
	"sea-battle-2/internal/game"
)

type Difficulty string

const (
	Easy   Difficulty = "easy"
	Medium Difficulty = "medium"
	Hard   Difficulty = "hard"
)

func (d Difficulty) Valid() bool { return d == Easy || d == Medium || d == Hard }
func (d Difficulty) Default() Difficulty {
	if d == "" {
		return Medium
	}
	return d
}

// Decide never receives a fleet. Cancellation also interrupts expensive inference.
func Decide(ctx context.Context, d Difficulty, k Knowledge, rng *rand.Rand) (game.Coord, bool) {
	if ctx.Err() != nil {
		return game.Coord{}, false
	}
	switch d.Default() {
	case Easy:
		return easy(k, rng)
	case Hard:
		return hard(ctx, k, rng)
	default:
		return Choose(k, rng)
	}
}
func easy(k Knowledge, rng *rand.Rand) (game.Coord, bool) {
	// Follow every live hit until it sinks. No orientation/density optimization:
	// randomly choose among viable neighboring targets, otherwise search at random.
	viable := [100]bool{}
	for id, length := range game.Lengths {
		if k.Sunk[id] {
			continue
		}
		for n := 0; n < 200; n++ {
			cells := game.Cells(game.Placement{Coord: game.Coord{X: n % 10, Y: n / 10 % 10}, Horizontal: n >= 100}, length)
			valid, hit, unknown := true, false, false
			for _, c := range cells {
				if !c.Valid() || k.Shots[c.Index()] == game.Miss || k.Shots[c.Index()] == game.Sunk {
					valid = false
					break
				}
				hit = hit || k.Shots[c.Index()] == game.Hit
				unknown = unknown || k.Shots[c.Index()] == game.Unknown
			}
			if !valid || !hit || !unknown {
				continue
			}
			for _, c := range cells {
				if k.Shots[c.Index()] == game.Unknown {
					viable[c.Index()] = true
				}
			}
		}
	}
	targets, unknown := []int{}, []int{}
	for n, m := range k.Shots {
		if m != game.Unknown {
			continue
		}
		unknown = append(unknown, n)
		if !viable[n] {
			continue
		}
		x, y := n%10, n/10
		for _, v := range []game.Coord{{X: x - 1, Y: y}, {X: x + 1, Y: y}, {X: x, Y: y - 1}, {X: x, Y: y + 1}} {
			if v.Valid() && k.Shots[v.Index()] == game.Hit {
				targets = append(targets, n)
				break
			}
		}
	}
	if len(targets) == 0 {
		targets = unknown
	}
	if len(targets) == 0 {
		return game.Coord{}, false
	}
	n := targets[rng.IntN(len(targets))]
	return game.Coord{X: n % 10, Y: n / 10}, true
}
