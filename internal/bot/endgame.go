package bot

import (
	"context"
	"encoding/binary"
	"math"
	"math/rand/v2"
	"sea-battle-2/internal/game"
	"sort"
)

// With one hull left and a small hypothesis set, solve the adaptive decision
// tree exactly: minimize expected additional shots, including both hit and miss
// branches. This is stronger than simply maximizing the next shot's hit chance.
// A deterministic work budget bounds cost; callers discard incomplete searches.
func optimalEndgame(ctx context.Context, p posterior, rng *rand.Rand) (game.Coord, bool) {
	if len(p.candidates) != 1 {
		return game.Coord{}, false
	}
	states := []mask{}
	for _, m := range p.candidates[0] {
		if m.contains(p.hits) {
			states = append(states, m.and(p.unknown))
		}
	}
	if len(states) == 0 || len(states) > 18 {
		return game.Coord{}, false
	}
	memo := map[string]float64{}
	work := 0
	aborted := false
	key := func(states []mask) string {
		// Filtering preserves candidate order, so no sorting or mutation is needed.
		b := make([]byte, len(states)*16)
		for i, m := range states {
			binary.LittleEndian.PutUint64(b[i*16:], m.lo)
			binary.LittleEndian.PutUint64(b[i*16+8:], m.hi)
		}
		return string(b)
	}
	var solve func([]mask) float64
	solve = func(states []mask) float64 {
		if len(states) == 0 {
			return 0
		}
		work++
		if work > 40000 || (work%128 == 0 && ctx.Err() != nil) {
			aborted = true
			return 0
		}
		common, union := states[0], mask{}
		for _, m := range states {
			common = common.and(m)
			union = union.or(m)
		}
		if union.empty() {
			return 0
		}
		// Guaranteed cells cost one shot each and provide no information. Factoring
		// them out collapses permutations of equivalent certain-hit actions.
		if !common.empty() {
			next := make([]mask, len(states))
			for i, m := range states {
				next[i] = m.without(common)
			}
			return float64(common.count()) + solve(next)
		}
		k := key(states)
		if v, ok := memo[k]; ok {
			return v
		}
		scores := [100]int{}
		for _, m := range states {
			m.addTo(&scores)
		}
		choices := []int{}
		for i, n := range scores {
			if n > 0 {
				choices = append(choices, i)
			}
		}
		sort.SliceStable(choices, func(i, j int) bool { return scores[choices[i]] > scores[choices[j]] })
		best := math.Inf(1)
		for _, i := range choices {
			cell := bit(i)
			hit, miss := []mask{}, []mask{}
			for _, m := range states {
				if !m.and(cell).empty() {
					hit = append(hit, m.without(cell))
				} else {
					miss = append(miss, m)
				}
			}
			value := 1 + (float64(len(hit))*solve(hit)+float64(len(miss))*solve(miss))/float64(len(states))
			if aborted {
				return 0
			}
			best = min(best, value)
		}
		memo[k] = best
		return best
	}
	common := states[0]
	for _, m := range states {
		common = common.and(m)
	}
	scores := [100]int{}
	for _, m := range states {
		m.addTo(&scores)
	}
	best := math.Inf(1)
	choices := []int{}
	for i, n := range scores {
		if n == 0 {
			continue
		}
		if !common.empty() {
			if !common.and(bit(i)).empty() {
				choices = append(choices, i)
			}
			continue
		}
		cell := bit(i)
		hit, miss := []mask{}, []mask{}
		for _, m := range states {
			if !m.and(cell).empty() {
				hit = append(hit, m.without(cell))
			} else {
				miss = append(miss, m)
			}
		}
		value := 1 + (float64(len(hit))*solve(hit)+float64(len(miss))*solve(miss))/float64(len(states))
		if aborted || ctx.Err() != nil {
			return game.Coord{}, false
		}
		if value < best-1e-9 {
			best = value
			choices = choices[:0]
		}
		if math.Abs(value-best) < 1e-9 {
			choices = append(choices, i)
		}
	}
	if len(choices) == 0 {
		return game.Coord{}, false
	}
	i := choices[rng.IntN(len(choices))]
	return game.Coord{X: i % 10, Y: i / 10}, true
}
