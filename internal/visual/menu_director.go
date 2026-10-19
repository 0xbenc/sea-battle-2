package visual

import (
	"math"
	"math/rand/v2"
)

type idleKind uint8

const (
	idleNone idleKind = iota
	idleDogfight
	idleBombing
	idleSub
	idleResupply
	idleFlares
	idleLanding
	idleBattery
	idleLeviathan
	idleCIWS
	idleShore
	idleRadar
	idleDefense
)

var storyLengths = [...]float64{0, 58, 58, 19, 28, 16, 17, 18, 13, 17}

type idleCue struct {
	kind    idleKind
	age     float64
	variant uint64
}

// MenuIdle is a read-only rendering snapshot. Rendering never draws randomness
// or advances the director, so extra View calls and resizing cannot reroll events.
type MenuIdle struct {
	story, ambient idleCue
	sheen          float64
}

type idleSlot struct {
	kind            idleKind
	start, duration float64
	variant         uint64
}

func (s idleSlot) end() float64 { return s.start + s.duration }
func (s idleSlot) cue(t float64) idleCue {
	if t < s.start || t >= s.end() {
		return idleCue{}
	}
	return idleCue{s.kind, t - s.start, s.variant}
}

// MenuDirector owns bounded scheduling state and a session-seeded random stream.
// There is no fixed playlist or master period. Reproducible seeds are only used
// by captures/tests; play supplies a fresh runtime-random seed at startup.
type MenuDirector struct {
	rng                   *rand.Rand
	now                   float64
	story, ambient, sheen idleSlot
	storyBag              []idleKind
}

func NewMenuDirector(seed uint64) *MenuDirector {
	d := &MenuDirector{rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))}
	d.nextStory(d.between(5, 10))
	d.nextAmbient(d.between(28, 65))
	d.sheen = idleSlot{start: d.between(24, 50), duration: 2.2}
	d.separateAmbient()
	return d
}
func (d *MenuDirector) between(lo, hi float64) float64 { return lo + d.rng.Float64()*(hi-lo) }
func idleFamily(k idleKind) idleKind {
	if k == idleBombing {
		return idleDogfight
	}
	return k
}
func (d *MenuDirector) nextStory(start float64) {
	if len(d.storyBag) == 0 {
		d.storyBag = []idleKind{idleDogfight, idleBombing, idleSub, idleResupply, idleFlares, idleLanding, idleBattery, idleLeviathan, idleCIWS}
		// Shuffle whole rounds until neighboring sorties are separated, including
		// the boundary with the previous round. Every event still appears once.
		for {
			d.rng.Shuffle(len(d.storyBag), func(i, j int) {
				d.storyBag[i], d.storyBag[j] = d.storyBag[j], d.storyBag[i]
			})
			previous, valid := d.story.kind, true
			for _, k := range d.storyBag {
				if idleFamily(k) == idleFamily(previous) {
					valid = false
					break
				}
				previous = k
			}
			if valid {
				break
			}
		}
	}
	k := d.storyBag[0]
	d.storyBag = d.storyBag[1:]
	d.story = idleSlot{k, start, storyLengths[k], d.rng.Uint64()}
}
func (d *MenuDirector) nextAmbient(start float64) {
	choices := []idleKind{idleShore, idleRadar, idleDefense}
	d.ambient = idleSlot{choices[d.rng.IntN(len(choices))], start, 5, d.rng.Uint64()}
}
func (d *MenuDirector) separateAmbient() {
	if d.ambient.start < d.story.end() && d.ambient.end() > d.story.start {
		d.ambient.start = d.story.end() + d.between(7, 14)
	}
}
func (d *MenuDirector) Advance(dt float64) {
	if d == nil || dt <= 0 || math.IsNaN(dt) || math.IsInf(dt, 0) {
		return
	}
	target := d.now + dt
	// Process boundaries in chronological order even after a long time jump.
	for {
		next := math.Min(d.story.end(), math.Min(d.ambient.end(), d.sheen.end()))
		if next > target {
			break
		}
		d.now = next
		if d.story.end() <= next {
			d.nextStory(next + d.between(10, 20))
		}
		if d.ambient.end() <= next {
			d.nextAmbient(next + d.between(35, 75))
		}
		if d.sheen.end() <= next {
			d.sheen = idleSlot{start: next + d.between(35, 75), duration: 2.2}
		}
		d.separateAmbient()
	}
	d.now = target
}
func (d *MenuDirector) Snapshot() MenuIdle {
	if d == nil {
		return MenuIdle{sheen: -1}
	}
	sheen := -1.0
	if d.now >= d.sheen.start && d.now < d.sheen.end() {
		sheen = d.now - d.sheen.start
	}
	return MenuIdle{d.story.cue(d.now), d.ambient.cue(d.now), sheen}
}
func menuIdleAt(t float64) MenuIdle {
	d := NewMenuDirector(1)
	d.Advance(t)
	return d.Snapshot()
}
func (c idleCue) sortie() sortieState {
	return sortieState{age: c.age, win: c.kind == idleBombing || c.variant%100 < 62, high: (c.variant>>8)&1 == 1, bombing: c.kind == idleBombing}
}
func (c idleCue) flight() bool { return c.kind == idleDogfight || c.kind == idleBombing }
func (c idleCue) onDeck() bool {
	s := c.sortie()
	// The entire landing approach passes on the near side of the island.
	return s.age < sortieLiftTime+5 || (s.win && s.age >= sortieLiftTime+39)
}

func (r *Raster) menuBackground(m MenuIdle) {
	c := m.story
	switch c.kind {
	case idleDogfight, idleBombing:
		if !c.onDeck() {
			r.idleSortie(c.sortie())
		}
	case idleSub:
		r.idleMissileSub(c.age)
	case idleLeviathan:
		r.idleLeviathan(c.age)
	case idleLanding:
		r.idleBadLanding(c.age)
	case idleBattery:
		r.idleAwakening(c.age)
	case idleFlares:
		r.idleFlareEscape(c.age)
	case idleCIWS:
		r.idleCIWS(c.age, false)
	}
	if m.ambient.kind == idleShore {
		r.idleShore(m.ambient.age, int(m.ambient.variant))
	}
}
func (r *Raster) menuForeground(m MenuIdle) {
	c := m.story
	if c.flight() {
		if c.onDeck() {
			r.idleSortie(c.sortie())
		}
	}
	if c.kind == idleCIWS {
		r.idleCIWS(c.age, true)
	}
	if c.kind == idleResupply {
		r.idleResupply(c.age)
	}
	switch m.ambient.kind {
	case idleRadar:
		r.idleCarrierCue(m.ambient.age, true)
	case idleDefense:
		r.idleCarrierCue(m.ambient.age, false)
	}
}
