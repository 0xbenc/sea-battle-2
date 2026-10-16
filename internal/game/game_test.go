package game

import (
	"reflect"
	"testing"
)

func TestPlacementAndRejectionAreAtomic(t *testing.T) {
	g := New(0)
	f := RandomFleet(3)
	if err := g.SetFleet(0, f); err != nil {
		t.Fatal(err)
	}
	f[0] = f[1]
	before := g.View(1)
	if g.SetFleet(1, f) == nil {
		t.Fatal("overlap accepted")
	}
	if !reflect.DeepEqual(before, g.View(1)) {
		t.Fatal("rejected fleet changed state")
	}
	for i := uint64(0); i < 1000; i++ {
		if err := ValidateFleet(RandomFleet(i)); err != nil {
			t.Fatal(i, err)
		}
	}
}
func TestTurnsPrivacySinkingVictoryAndRematch(t *testing.T) {
	g := New(0)
	f0, f1 := RandomFleet(1), RandomFleet(2)
	g.SetFleet(0, f0)
	g.SetFleet(1, f1)
	before := g.View(0)
	if g.Fire(1, Coord{0, 0}, 0) == nil {
		t.Fatal("out of turn accepted")
	}
	if g.Fire(0, Coord{10, 0}, 0) == nil {
		t.Fatal("outside board accepted")
	}
	if !reflect.DeepEqual(before, g.View(0)) {
		t.Fatal("illegal shot mutated game")
	}
	target := Cells(f1[0], 2)[0]
	if err := g.Fire(0, target, 0); err != nil {
		t.Fatal(err)
	}
	v := g.View(0)
	if len(v.Own) != 5 || v.Events[0].Ship != -1 || !v.Events[0].Hit {
		t.Fatal("hit leaked unsunk identity or lost result")
	}
	if g.View(1).Events[0].Ship != 0 {
		t.Fatal("defender cannot identify own ship")
	}
	// Opponent deliberately misses while player zero resolves all seventeen squares.
	misses := []Coord{}
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			c := Coord{x, y}
			occupied := false
			for id, p := range f0 {
				for _, s := range Cells(p, Lengths[id]) {
					if c == s {
						occupied = true
					}
				}
			}
			if !occupied {
				misses = append(misses, c)
			}
		}
	}
	mi := 0
	for id, p := range f1 {
		for _, c := range Cells(p, Lengths[id]) {
			if c == target {
				continue
			}
			if err := g.Fire(1, misses[mi], len(g.Events)); err != nil {
				t.Fatal(err)
			}
			mi++
			if err := g.Fire(0, c, len(g.Events)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if g.Phase != "finished" || g.Winner != 0 {
		t.Fatal("did not finish")
	}
	for _, m := range g.View(0).Outgoing {
		if m == Hit {
			t.Fatal("sunk hull left partial hit marks")
		}
	}
	if g.Fire(0, Coord{0, 0}, len(g.Events)) == nil {
		t.Fatal("shot after victory accepted")
	}
	g.RequestRematch(0)
	if g.Phase != "finished" {
		t.Fatal("unilateral rematch")
	}
	g.RequestRematch(1)
	if g.Match != 2 || g.Phase != "placement" || len(g.Events) != 0 || g.Ready[0] {
		t.Fatal("bad rematch reset")
	}
}
func TestDuplicateAndStaleShotDoNotConsumeTurn(t *testing.T) {
	g := New(0)
	g.SetFleet(0, RandomFleet(1))
	g.SetFleet(1, RandomFleet(2))
	g.Fire(0, Coord{0, 0}, 0)
	g.Fire(1, Coord{0, 0}, 1)
	before := g.View(0)
	if g.Fire(0, Coord{0, 0}, 2) == nil || g.Fire(0, Coord{1, 0}, 0) == nil {
		t.Fatal("accepted duplicate/stale shot")
	}
	if !reflect.DeepEqual(before, g.View(0)) {
		t.Fatal("rejection changed state")
	}
}
