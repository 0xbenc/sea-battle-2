package visual

import (
	"fmt"
	"reflect"
	"sea-battle-2/internal/game"
	"strings"
	"testing"
)

func TestEightResultScenes(t *testing.T) {
	for _, sd := range []bool{true, false} {
		seen := map[string]bool{}
		for player := 0; player < 2; player++ {
			for _, win := range []bool{true, false} {
				for _, dominant := range []bool{false, true} {
					b := combatFixture(player, sd)
					b.State.Phase = "finished"
					b.State.Winner = player
					if !win {
						b.State.Winner = 1 - player
					}
					b.State.Incoming, b.State.Outgoing = [100]game.Mark{}, [100]game.Mark{}
					hp := 4
					if dominant {
						hp = 9
					}
					marks := &b.State.Incoming
					if !win {
						marks = &b.State.Outgoing
					}
					for i := 0; i < 17-hp; i++ {
						marks[i] = game.Hit
					}
					name := fmt.Sprintf("result-win%t-dominant%t", win, dominant)
					var first *Frame
					for _, at := range []float64{0, 3, 8, 20} {
						b.ResultTime = at
						f := RenderBattle(b)
						plain := f.Plain()
						if !strings.Contains(plain, fmt.Sprintf("%d/17 HULL INTACT", hp)) || strings.Contains(plain, "HOSTILE WATERS") {
							t.Fatal("bad result layout", name, plain)
						}
						if strings.Contains(plain, "OVERWHELMING") != dominant {
							t.Fatal("wrong victory tier")
						}
						if first == nil {
							first = f
						} else if reflect.DeepEqual(first, f) {
							t.Fatal("static result")
						}
						capture(t, f, fmt.Sprintf("%s-%02.0f", name, at), sd, player)
					}
					key := fmt.Sprint(first.Cells)
					if seen[key] {
						t.Fatal("duplicate result scene")
					}
					seen[key] = true
					b.ResultCharts = true
					if !strings.Contains(RenderBattle(b).Plain(), "HOSTILE WATERS") {
						t.Fatal("charts unavailable")
					}
				}
			}
		}
	}
}
