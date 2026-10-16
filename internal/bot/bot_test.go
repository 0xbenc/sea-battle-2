package bot

import (
	"math/rand/v2"
	"sea-battle-2/internal/game"
	"testing"
)

func random(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, seed+1)) }

func TestTargetKnowledgeAndTouchingShips(t *testing.T) {
	k := Knowledge{}
	k.Shots[44] = game.Hit
	for seed := uint64(0); seed < 100; seed++ {
		at, ok := Choose(k, random(seed))
		if !ok {
			t.Fatal("no target")
		}
		if at.Index() != 34 && at.Index() != 43 && at.Index() != 45 && at.Index() != 54 {
			t.Fatalf("ignored isolated hit: %v", at)
		}
	}
	// An unsunk hull may touch a sunk hull, including along its full length.
	k = Knowledge{Sunk: [5]bool{true}}
	for i := range k.Shots {
		k.Shots[i] = game.Miss
	}
	k.Shots[0], k.Shots[1] = game.Sunk, game.Sunk
	k.Shots[10], k.Shots[11], k.Shots[12] = game.Hit, game.Hit, game.Unknown
	at, ok := Choose(k, random(1))
	if !ok || at.Index() != 12 {
		t.Fatal("assumed a border around sunk ship", at)
	}
	k.Shots[12] = game.Sunk
	if _, ok := Choose(k, random(1)); ok {
		t.Fatal("selected tried cell")
	}
}
func TestObservationExcludesPrivateInformation(t *testing.T) {
	g := game.New(0)
	g.SetFleet(0, game.RandomFleet(1))
	g.SetFleet(1, game.RandomFleet(2))
	v := g.View(0)
	original := Observe(v)
	v.Own = game.RandomFleet(900)
	v.Incoming[15] = game.Hit
	v.Events = []game.Event{{Player: 1, Hit: true, Sunk: true, Ship: 4}}
	if Observe(v) != original {
		t.Fatal("opponent or private information affects targeting")
	}
	v.Events = append(v.Events, game.Event{Player: 0, Hit: true, Sunk: true, Ship: 1})
	k := Observe(v)
	if !k.Sunk[1] || k.Sunk[2] {
		t.Fatal("lost duplicate length-three hull")
	}
}
func TestSeededMatchesAndRandomBaseline(t *testing.T) {
	wins, totalShots := 0, 0
	const games = 500
	for seed := uint64(0); seed < games; seed++ {
		g := game.New(int(seed % 2))
		g.SetFleet(0, game.RandomFleet(seed*2))
		g.SetFleet(1, game.RandomFleet(seed*2+1))
		rng := random(seed + 5000)
		for g.Phase == "battle" {
			p := g.Turn
			v := g.View(p)
			var at game.Coord
			if p == 0 {
				var ok bool
				at, ok = Choose(Observe(v), rng)
				if !ok {
					t.Fatal("no target")
				}
				totalShots++
			} else {
				legal := []int{}
				for i, m := range v.Outgoing {
					if m == game.Unknown {
						legal = append(legal, i)
					}
				}
				n := legal[rng.IntN(len(legal))]
				at = game.Coord{X: n % 10, Y: n / 10}
			}
			if err := g.Fire(p, at, len(g.Events)); err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}
			if len(g.Events) > 199 {
				t.Fatal("unbounded match")
			}
		}
		if g.Winner == 0 {
			wins++
		}
	}
	t.Logf("bot vs random: %d/%d wins; mean bot shots %.2f", wins, games, float64(totalShots)/games)
	if wins < games*3/4 {
		t.Fatal("bot fails broad random baseline")
	}
}
