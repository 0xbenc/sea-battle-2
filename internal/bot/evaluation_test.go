package bot

import (
	"context"
	"os"
	"sea-battle-2/internal/game"
	"strconv"
	"testing"
	"time"
)

// Common hidden boards and independent targeting seeds make difficulty changes
// reviewable. Opt into a larger holdout run without slowing every unit-test run.
func TestDifficultyEvaluation(t *testing.T) {
	games, offset := 24, 8000
	if s := os.Getenv("SB2_BOT_EVAL_GAMES"); s != "" {
		n, e := strconv.Atoi(s)
		if e != nil || n < 24 {
			t.Fatal("evaluation needs at least 24 games")
		}
		games = n
	}
	if s := os.Getenv("SB2_BOT_EVAL_SEED"); s != "" {
		n, e := strconv.Atoi(s)
		if e != nil {
			t.Fatal(e)
		}
		offset = n
	}
	totals := map[Difficulty]int{}
	results := map[Difficulty][]int{}
	for _, d := range []Difficulty{Easy, Medium, Hard} {
		start := time.Now()
		maxThink := time.Duration(0)
		for seed := 0; seed < games; seed++ {
			g := game.New(0)
			g.SetFleet(0, game.RandomFleet(900))
			g.SetFleet(1, game.RandomFleet(uint64(seed+offset)))
			rng := random(uint64(seed + offset + 4000))
			for g.Phase == "battle" {
				g.Turn = 0
				begin := time.Now()
				at, ok := Decide(context.Background(), d, Observe(g.View(0)), rng)
				maxThink = max(maxThink, time.Since(begin))
				if !ok {
					t.Fatal(d, seed, "no target")
				}
				if err := g.Fire(0, at, len(g.Events)); err != nil {
					t.Fatal(d, seed, err)
				}
			}
			totals[d] += len(g.Events)
			results[d] = append(results[d], len(g.Events))
		}
		t.Logf("%s: %.2f mean shots to clear %d boards; max decision %s; elapsed %s", d, float64(totals[d])/float64(games), games, maxThink, time.Since(start))
	}
	// Pair different hidden boards and alternate the opening turn. Shot counts
	// determine these races exactly: neither opponent sees incoming-shot knowledge.
	for _, pair := range [][2]Difficulty{{Hard, Medium}, {Hard, Easy}, {Medium, Easy}} {
		wins := 0
		for i, a := range results[pair[0]] {
			b := results[pair[1]][(i+1)%games]
			if a < b || (a == b && i%2 == 0) {
				wins++
			}
		}
		t.Logf("%s vs %s: %d/%d wins", pair[0], pair[1], wins, games)
	}
	if totals[Easy] <= totals[Medium] || totals[Medium] <= totals[Hard] {
		t.Fatal("difficulty ordering", totals)
	}
}

func TestDifficultiesOnTouchingCornerFleets(t *testing.T) {
	for _, d := range []Difficulty{Easy, Medium, Hard} {
		total := 0
		for transform := 0; transform < 8; transform++ {
			fleet := make([]game.Placement, 5)
			for i, l := range game.Lengths {
				x, y := 0, i
				if transform&1 != 0 {
					x = 10 - l
				}
				if transform&2 != 0 {
					y = 9 - y
				}
				horizontal := transform&4 == 0
				if !horizontal {
					x, y = y, x
				}
				fleet[i] = game.Placement{Coord: game.Coord{X: x, Y: y}, Horizontal: horizontal}
			}
			if err := game.ValidateFleet(fleet); err != nil {
				t.Fatal(err)
			}
			g := game.New(0)
			g.SetFleet(0, game.RandomFleet(1))
			g.SetFleet(1, fleet)
			rng := random(uint64(92000 + transform))
			for g.Phase == "battle" {
				g.Turn = 0
				at, ok := Decide(context.Background(), d, Observe(g.View(0)), rng)
				if !ok {
					t.Fatal(d, "no target")
				}
				if err := g.Fire(0, at, len(g.Events)); err != nil {
					t.Fatal(err)
				}
			}
			total += len(g.Events)
		}
		t.Logf("%s: %.2f mean shots across 8 touching corner fleets", d, float64(total)/8)
	}
}
