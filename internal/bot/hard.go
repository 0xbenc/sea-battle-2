package bot

import (
	"context"
	"math/bits"
	"math/rand/v2"
	"sea-battle-2/internal/game"
)

// Two machine words cover all 100 squares; fleet constraints stay allocation-free.
type mask struct{ lo, hi uint64 }

func bit(i int) mask {
	if i < 64 {
		return mask{lo: 1 << i}
	}
	return mask{hi: 1 << (i - 64)}
}
func (a mask) or(b mask) mask         { return mask{a.lo | b.lo, a.hi | b.hi} }
func (a mask) and(b mask) mask        { return mask{a.lo & b.lo, a.hi & b.hi} }
func (a mask) without(b mask) mask    { return mask{a.lo &^ b.lo, a.hi &^ b.hi} }
func (a mask) empty() bool            { return a.lo == 0 && a.hi == 0 }
func (a mask) contains(b mask) bool   { return b.without(a).empty() }
func (a mask) count() int             { return bits.OnesCount64(a.lo) + bits.OnesCount64(a.hi) }
func (a mask) addTo(scores *[100]int) { a.addWeight(scores, 1) }
func (a mask) addWeight(scores *[100]int, weight int) {
	for a.lo != 0 {
		i := bits.TrailingZeros64(a.lo)
		scores[i] += weight
		a.lo &= a.lo - 1
	}
	for a.hi != 0 {
		i := bits.TrailingZeros64(a.hi)
		scores[64+i] += weight
		a.hi &= a.hi - 1
	}
}

type posterior struct {
	candidates    [][]mask
	hits, unknown mask
}

func possibilities(k Knowledge) posterior {
	p := posterior{}
	var blocked mask
	for i, m := range k.Shots {
		switch m {
		case game.Hit:
			p.hits = p.hits.or(bit(i))
		case game.Unknown:
			p.unknown = p.unknown.or(bit(i))
		default:
			blocked = blocked.or(bit(i))
		}
	}
	for id, length := range game.Lengths {
		if k.Sunk[id] {
			continue
		}
		candidates := []mask{}
		for n := 0; n < 200; n++ {
			cells := game.Cells(game.Placement{Coord: game.Coord{X: n % 10, Y: n / 10 % 10}, Horizontal: n >= 100}, length)
			m := mask{}
			valid := true
			for _, c := range cells {
				if !c.Valid() {
					valid = false
					break
				}
				m = m.or(bit(c.Index()))
			}
			// An unsunk hull cannot consist entirely of already-hit cells.
			if valid && m.and(blocked).empty() && !m.and(p.unknown).empty() {
				candidates = append(candidates, m)
			}
		}
		p.candidates = append(p.candidates, candidates)
	}
	return p
}

// Exact joint enumeration for the last one or two ships. Earlier turns use
// conditional Gibbs sampling of complete, nonoverlapping fleets, including all
// unresolved hits. The assumed prior is uniform over legal labeled placements.
// This estimates marginals, not a proof of globally optimal Battleship play.
func (p posterior) scores(ctx context.Context, rng *rand.Rand) ([100]int, [100]int, bool) {
	var hunt, target [100]int
	n := len(p.candidates)
	if n == 0 {
		return hunt, target, false
	}
	shortest := 5
	for _, cs := range p.candidates {
		if len(cs) > 0 {
			shortest = min(shortest, cs[0].count())
		}
	}
	observe := func(fleet []mask) {
		for _, m := range fleet {
			if m.count() == shortest {
				m.and(p.unknown).addTo(&hunt)
			}
			if !m.and(p.hits).empty() {
				m.and(p.unknown).addWeight(&target, 60/m.and(p.unknown).count())
			}
		}
	}
	fleet := make([]mask, n)
	found := false
	if n <= 2 {
		for _, a := range p.candidates[0] {
			if ctx.Err() != nil {
				return hunt, target, false
			}
			fleet[0] = a
			if n == 1 {
				if a.contains(p.hits) {
					observe(fleet)
					found = true
				}
				continue
			}
			for _, b := range p.candidates[1] {
				if a.and(b).empty() && a.or(b).contains(p.hits) {
					fleet[1] = b
					observe(fleet)
					found = true
				}
			}
		}
		return hunt, target, found
	}
	// Randomized bounded DFS initializes each chain. Prune any hit which no
	// unassigned candidate can explain; touching hulls are allowed throughout.
	visits := 0
	var search func(int, mask) bool
	search = func(depth int, occupied mask) bool {
		visits++
		if visits > 100000 || (visits%128 == 0 && ctx.Err() != nil) {
			return false
		}
		if depth == n {
			return occupied.contains(p.hits)
		}
		missing := p.hits.without(occupied)
		possible := mask{}
		for j := depth; j < n; j++ {
			for _, m := range p.candidates[j] {
				if m.and(occupied).empty() {
					possible = possible.or(m)
				}
			}
		}
		if !possible.contains(missing) {
			return false
		}
		for _, i := range rng.Perm(len(p.candidates[depth])) {
			m := p.candidates[depth][i]
			if m.and(occupied).empty() {
				fleet[depth] = m
				if search(depth+1, occupied.or(m)) {
					return true
				}
			}
		}
		return false
	}
	for chain := 0; chain < 4; chain++ {
		visits = 0
		if !search(0, mask{}) {
			return hunt, target, false
		}
		// Two-hull updates allow hit ownership to move between touching ships.
		// Reservoir sampling selects uniformly from each conditional distribution.
		for step := 0; step < 1024; step++ {
			if ctx.Err() != nil {
				return hunt, target, false
			}
			a := rng.IntN(n)
			b := a
			if step%12 == 0 {
				b = (a + 1 + rng.IntN(n-1)) % n
			}
			occupied := mask{}
			for i, m := range fleet {
				if i != a && i != b {
					occupied = occupied.or(m)
				}
			}
			required := p.hits.without(occupied)
			count := 0
			for _, ma := range p.candidates[a] {
				if !ma.and(occupied).empty() {
					continue
				}
				if a == b {
					if ma.contains(required) {
						count++
						if rng.IntN(count) == 0 {
							fleet[a] = ma
						}
					}
					continue
				}
				for _, mb := range p.candidates[b] {
					if mb.and(occupied.or(ma)).empty() && ma.or(mb).contains(required) {
						count++
						if rng.IntN(count) == 0 {
							fleet[a], fleet[b] = ma, mb
						}
					}
				}
			}
			if step >= 256 && step%2 == 0 {
				observe(fleet)
			}
		}
	}
	return hunt, target, true
}
func hard(ctx context.Context, k Knowledge, rng *rand.Rand) (game.Coord, bool) {
	p := possibilities(k)
	if at, ok := optimalEndgame(ctx, p, rng); ok {
		return at, true
	}
	var hunt, target [100]int
	ok := true
	if p.hits.empty() && len(p.candidates) > 2 {
		for _, cs := range p.candidates {
			for _, m := range cs {
				m.and(p.unknown).addTo(&hunt)
			}
		}
	} else {
		hunt, target, ok = p.scores(ctx, rng)
	}
	if ctx.Err() != nil {
		return game.Coord{}, false
	}
	if !ok {
		return Choose(k, rng)
	} // bounded inference fallback, still knowledge-only
	scores := hunt
	for _, s := range target {
		if s > 0 {
			scores = target
			break
		}
	}
	// Blend a local-search heuristic into the hunt only after multiple sunk
	// hulls reveal a dense cluster. An isolated sink is not enough evidence.
	// This uses revealed cells, never hidden placement.
	if p.hits.empty() {
		known := []game.Coord{}
		sunk := 0
		for _, s := range k.Sunk {
			if s {
				sunk++
			}
		}
		minX, minY, maxX, maxY := 10, 10, -1, -1
		for i, m := range k.Shots {
			if m == game.Sunk {
				c := game.Coord{X: i % 10, Y: i / 10}
				known = append(known, c)
				minX = min(minX, c.X)
				maxX = max(maxX, c.X)
				minY = min(minY, c.Y)
				maxY = max(maxY, c.Y)
			}
		}
		clustered := sunk >= 2 && maxX-minX < 6 && maxY-minY < 6 && len(known)*2 >= (maxX-minX+1)*(maxY-minY+1)
		for i := range scores {
			distance := 100
			for _, c := range known {
				dx, dy := i%10-c.X, i/10-c.Y
				if dx < 0 {
					dx = -dx
				}
				if dy < 0 {
					dy = -dy
				}
				distance = min(distance, dx+dy)
			}
			scores[i] *= 10
			if clustered && distance <= 2 {
				scores[i] *= 4
			}
		}
	}
	best, choices := -1, []int{}
	for i, s := range scores {
		if k.Shots[i] != game.Unknown {
			continue
		}

		if s > best {
			best = s
			choices = choices[:0]
		}
		if s == best {
			choices = append(choices, i)
		}
	}
	if len(choices) == 0 {
		return Choose(k, rng)
	}
	i := choices[rng.IntN(len(choices))]
	return game.Coord{X: i % 10, Y: i / 10}, true
}
