package visual

import (
	"sea-battle-2/internal/game"
	"strings"
	"testing"
)

func TestPlacementLayoutAndStages(t *testing.T) {
	for _, sd := range []bool{true, false} {
		l := PlacementGeometry(sd)
		for y := 0; y < 10; y++ {
			for x := 0; x < 10; x++ {
				c, ok := PlacementCellAt(sd, l.X+x*l.CW+l.CW/2, (l.Y+y*l.CH+l.CH/2)/2)
				if !ok || c != (game.Coord{X: x, Y: y}) {
					t.Fatalf("SD=%v inaccessible cell %d,%d: %v", sd, x, y, c)
				}
			}
		}
		for team := 0; team < 2; team++ {
			for _, stage := range []string{"empty", "partial", "blocked", "ready", "moving", "locked", "help"} {
				v := game.New(0).View(team)
				v.Connected = [2]bool{true, true}
				b := BattlePresentation{State: v, Fleet: game.EmptyFleet(), SD: sd, Selected: 0, Solo: true, Difficulty: "medium"}
				switch stage {
				case "partial":
					b.Fleet[0] = game.Placement{Coord: game.Coord{X: 1, Y: 1}}
					b.Selected = 1
					b.Cursor = game.Coord{X: 3, Y: 2}
				case "blocked":
					b.Cursor = game.Coord{X: 9, Y: 9}
					b.Selected = 4
				case "ready":
					b.Fleet = game.RandomFleet(3)
					b.Selected = -1
					b.CanUndo = true
				case "moving":
					b.Fleet = game.RandomFleet(3)
					b.Selected = 4
					b.Cursor = game.Coord{X: 2, Y: 2}
				case "locked":
					b.State.Ready[team] = true
					b.State.Own = game.RandomFleet(3)
				case "help":
					b.PlacementHelp = true
				}
				f := RenderBattle(b)
				text := f.Plain()
				if strings.Contains(text, "HOSTILE") || strings.Contains(text, "SHOT 001") {
					t.Fatal("combat UI leaked into placement")
				}
				if stage == "ready" && !strings.Contains(text, "Fleet ready.") {
					t.Fatal("ready coaching missing")
				}
				if stage == "locked" && !strings.Contains(text, "FLEET LOCKED") {
					t.Fatal("locked state missing")
				}
				capture(t, f, "placement-new-"+stage, sd, team)
			}
		}
	}
}
