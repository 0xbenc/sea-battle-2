package visual

import (
	"reflect"
	"sea-battle-2/internal/game"
	"strings"
	"testing"
)

func TestImpactShapesStayInCell(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {3, 2}, {7, 5}} {
		var previous [][]RGB
		for _, mark := range []game.Mark{game.Miss, game.Hit, game.Sunk} {
			r := New(12, 6)
			r.combatImpact(2, 2, size[0], size[1], mark, 0)
			visible := false
			for i, pixel := range r.Pixels {
				if pixel == 0 {
					continue
				}
				visible = true
				x, y := i%r.W, i/r.W
				if x < 2 || y < 2 || x >= 2+size[0] || y >= 2+size[1] {
					t.Fatal("impact spills into another cell", size, mark)
				}
			}
			if !visible || len(r.Texts) != 0 {
				t.Fatal("impact must be visible pixel art", size, mark)
			}
			for _, other := range previous {
				if reflect.DeepEqual(other, r.Pixels) {
					t.Fatal("outcomes must remain distinct", size, mark)
				}
			}
			previous = append(previous, r.Pixels)
		}
	}
}

func TestImpactBoilAndSoloFade(t *testing.T) {
	for _, sd := range []bool{true, false} {
		b := combatFixture(0, sd)
		for _, mark := range []game.Mark{game.Miss, game.Hit} {
			a, c := New(12, 6), New(12, 6)
			a.combatImpact(2, 2, 7, 5, mark, 0)
			c.combatImpact(2, 2, 7, 5, mark, 1.5)
			if reflect.DeepEqual(a.Pixels, c.Pixels) {
				t.Fatal("impact does not animate")
			}
			for i := range a.Pixels {
				if (a.Pixels[i] == 0) != (c.Pixels[i] == 0) {
					t.Fatal("boil changes the impact footprint")
				}
			}
		}
		b.Handoff = true
		b.Clip = &game.Event{Player: 1, At: game.Coord{X: 2, Y: 3}, Ship: -1}
		for _, progress := range []float64{0, .25, .5, .75, 1} {
			b.HandoffProgress = progress
			f := RenderBattle(b)
			if strings.Contains(f.Plain(), "ENEMY TAKING AIM") {
				t.Fatal("old modal during fade")
			}
			if progress == .5 {
				for _, c := range f.Cells {
					if c.FG != 0 || c.BG != 0 {
						t.Fatal("transition midpoint must be black")
					}
				}
			}
			if progress == 1 {
				b.Handoff = false
				if !reflect.DeepEqual(f, RenderBattle(b)) {
					t.Fatal("fade must end on the incoming cinematic without a jump")
				}
			}
		}
	}
}

func combatFixture(team int, sd bool) BattlePresentation {
	g := game.New(0)
	g.SetFleet(0, game.RandomFleet(71))
	g.SetFleet(1, game.RandomFleet(72))
	v := g.View(team)
	v.Turn = team
	v.Connected = [2]bool{true, true}
	v.Outgoing[31] = game.Miss
	v.Outgoing[42] = game.Hit
	for i := 0; i < 5; i++ {
		v.Outgoing[i] = game.Sunk
	}
	for _, c := range game.Cells(v.Own[0], 2) {
		v.Incoming[c.Index()] = game.Sunk
	}
	v.Incoming[99] = game.Miss
	v.Events = []game.Event{{Player: team, At: game.Coord{X: 4, Y: 0}, Hit: true, Sunk: true, Ship: 4}, {Player: 1 - team, At: v.Own[0].Coord, Hit: true, Sunk: true, Ship: 0}, {Player: team, At: game.Coord{X: 1, Y: 3}, Ship: -1}, {Player: 1 - team, At: game.Coord{X: 9, Y: 9}, Ship: -1}}
	return BattlePresentation{State: v, SD: sd, Solo: true, Difficulty: "medium", Cursor: game.Coord{X: 5, Y: 6}, OwnCursor: v.Own[1].Coord}
}
func TestCombatStatesAndHiddenContacts(t *testing.T) {
	for _, sd := range []bool{true, false} {
		for team := 0; team < 2; team++ {
			for _, stage := range []string{"aim", "resolved", "enemy", "pending", "fleet", "victory", "defeat", "offline", "help"} {
				b := combatFixture(team, sd)
				switch stage {
				case "resolved":
					b.Cursor = game.Coord{X: 1, Y: 3}
				case "enemy":
					b.State.Turn = 1 - team
				case "pending":
					b.Pending = true
				case "fleet":
					b.InspectOwn = true
				case "victory":
					b.State.Phase = "finished"
					b.State.Winner = team
				case "defeat":
					b.State.Phase = "finished"
					b.State.Winner = 1 - team
				case "offline":
					b.State.Connected[1-team] = false
				case "help":
					b.CombatHelp = true
				}
				f := RenderBattle(b)
				if strings.Contains(f.Plain(), "DIRECTOR") || strings.Contains(f.Plain(), "AUTHORIZATION") {
					t.Fatal("old console text leaked")
				}
				if stage == "resolved" && combatCanFire(b) {
					t.Fatal("resolved cell can fire")
				}
				if (stage == "pending" || stage == "enemy" || stage == "offline" || stage == "fleet") && combatCanFire(b) {
					t.Fatal("unsafe fire affordance")
				}
				capture(t, f, "combat-new-"+stage, sd, team)
			}
			b := combatFixture(team, sd)
			first := RenderBattle(b)
			b.State.Own = game.RandomFleet(934)
			second := RenderBattle(b)
			grid, _ := CombatBoards(sd, false)
			for y := grid.Y / 2; y < (grid.Y+10*grid.CH)/2; y++ {
				for x := grid.X; x < grid.X+10*grid.CW; x++ {
					if first.Cells[y*first.W+x] != second.Cells[y*second.W+x] {
						t.Fatal("enemy chart depends on a fleet rather than confirmed marks")
					}
				}
			}
		}
	}
}
func TestCombatBoardCoordinates(t *testing.T) {
	for _, sd := range []bool{true, false} {
		for _, inspect := range []bool{true, false} {
			target, own := CombatBoards(sd, inspect)
			for _, board := range []CombatBoard{target, own} {
				for y := 0; y < 10; y++ {
					for x := 0; x < 10; x++ {
						c, ok := board.CellAt(board.X+x*board.CW+board.CW/2, (board.Y+y*board.CH+board.CH/2)/2)
						if !ok || c != (game.Coord{X: x, Y: y}) {
							t.Fatal("unreachable combat cell", sd, inspect, x, y, c)
						}
					}
				}
			}
		}
	}
}
