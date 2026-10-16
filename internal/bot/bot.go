// Package bot chooses targets using only information a shooter has earned.
package bot

import (
	"math/rand/v2"
	"sea-battle-2/internal/game"
)

// Knowledge deliberately excludes fleets, incoming shots and placement seeds.
type Knowledge struct {
	Shots [100]game.Mark
	Sunk  [5]bool
}

func Observe(v game.View) Knowledge {
	k := Knowledge{Shots: v.Outgoing}
	for _, e := range v.Events {
		if e.Player == v.Player && e.Sunk && e.Ship >= 0 && e.Ship < 5 {
			k.Sunk[e.Ship] = true
		}
	}
	return k
}

func Choose(k Knowledge, rng *rand.Rand) (game.Coord, bool) {
	var hunt, target [100]int
	for id, length := range game.Lengths {
		if k.Sunk[id] {
			continue
		}
		for n := 0; n < 200; n++ {
			cells := game.Cells(game.Placement{Coord: game.Coord{X: n % 10, Y: (n / 10) % 10}, Horizontal: n >= 100}, length)
			valid, hits := true, 0
			for _, c := range cells {
				if !c.Valid() || k.Shots[c.Index()] == game.Miss || k.Shots[c.Index()] == game.Sunk {
					valid = false
					break
				}
				if k.Shots[c.Index()] == game.Hit {
					hits++
				}
			}
			if !valid {
				continue
			}
			for _, c := range cells {
				if k.Shots[c.Index()] == game.Unknown {
					hunt[c.Index()]++
					if hits > 0 {
						target[c.Index()] += hits * hits
					}
				}
			}
		}
	}
	scores := hunt
	targeting := false
	for _, score := range target {
		if score > 0 {
			scores = target
			targeting = true
			break
		}
	}
	if !targeting {
		// Search among plausible squares; eliminate cells no remaining hull can
		// occupy. Systematic coverage and fleet-wide probability belong to Hard.
		choices := []int{}
		for i, m := range k.Shots {
			if m == game.Unknown && hunt[i] > 0 {
				choices = append(choices, i)
			}
		}
		if len(choices) > 0 {
			n := choices[rng.IntN(len(choices))]
			return game.Coord{X: n % 10, Y: n / 10}, true
		}
	}
	best, choices := -1, []int{}
	for i, mark := range k.Shots {
		if mark != game.Unknown {
			continue
		}
		if scores[i] > best {
			best = scores[i]
			choices = choices[:0]
		}
		if scores[i] == best {
			choices = append(choices, i)
		}
	}
	if len(choices) == 0 {
		return game.Coord{}, false
	}
	n := choices[rng.IntN(len(choices))]
	return game.Coord{X: n % 10, Y: n / 10}, true
}
