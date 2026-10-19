package visual

import (
	"fmt"
	"reflect"
	"sea-battle-2/internal/game"
	"testing"
)

func TestShotTakesAndPrivacy(t *testing.T) {
	for _, sd := range []bool{true, false} {
		for _, player := range []int{0, 1} {
			for _, outcome := range []string{"miss", "hit", "sunk"} {
				seen := map[int]bool{}
				for seq := 1; seq < 100 && len(seen) < 3; seq++ {
					e := game.Event{Seq: seq, Player: player, At: game.Coord{X: 3, Y: 4}, Ship: -1, Hit: outcome != "miss", Sunk: outcome == "sunk"}
					if e.Sunk {
						e.Ship = 4
					}
					take := shotTake(1, e)
					if seen[take] {
						continue
					}
					seen[take] = true
					b := combatFixture(player, sd)
					b.State.Match = 1
					b.Clip = &e
					for _, age := range []float64{.1, .6, 1.4, ShotDuration(e) - .2} {
						b.ClipTime = age
						frame := RenderBattle(b)
						if !reflect.DeepEqual(frame, RenderBattle(b)) {
							t.Fatal("take changes on redraw")
						}
						capture(t, frame, fmt.Sprintf("take-%d-%s-%04.1f", take, outcome, age), sd, player)
					}
					if outcome == "hit" {
						e.Ship = 0
						a := RenderBattle(b)
						e.Ship = 4
						c := RenderBattle(b)
						if !reflect.DeepEqual(a, c) {
							t.Fatal("unconfirmed enemy ship identity affects cinematic")
						}
					}
				}
				if len(seen) != 3 {
					t.Fatal("not all takes reachable")
				}
			}
		}
	}
}

func TestIncomingKnownHullTakes(t *testing.T) {
	for _, sd := range []bool{true, false} {
		for player := 0; player < 2; player++ {
			for ship := 0; ship < 5; ship++ {
				for seq := 1; seq <= 12; seq++ {
					b := combatFixture(player, sd)
					b.Clip = &game.Event{Seq: seq, Player: 1 - player, At: game.Coord{X: 2, Y: 3}, Ship: ship, Hit: true, Sunk: seq%2 == 0}
					b.ClipTime = 1.4
					f := RenderBattle(b)
					if f.W*f.H != len(f.Cells) {
						t.Fatal("invalid frame")
					}
					if ship == 4 {
						capture(t, f, fmt.Sprintf("incoming-%d-%v", shotTake(b.State.Match, *b.Clip), b.Clip.Sunk), sd, player)
					}
				}
			}
		}
	}
}
