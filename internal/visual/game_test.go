package visual

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sea-battle-2/internal/game"
	"strings"
	"testing"
)

func TestBattlePresentation(t *testing.T) {
	for _, sd := range []bool{true, false} {
		for player := 0; player < 2; player++ {
			g := game.New(0)
			g.SetFleet(0, game.RandomFleet(1))
			g.SetFleet(1, game.RandomFleet(2))
			b := BattlePresentation{State: g.View(player), Fleet: game.RandomFleet(3), SD: sd, Cursor: game.Coord{X: 9, Y: 9}}
			b.State.Connected = [2]bool{true, true}
			for _, phase := range []string{"placement", "battle", "finished"} {
				b.State.Phase = phase
				if phase == "placement" {
					b.State.Ready[player] = false
				}
				b.State.Winner = player
				f := RenderBattle(b)
				w, h := 160, 48
				if sd {
					w, h = 80, 24
				}
				if f.W != w || f.H != h || len(f.Cells) != w*h {
					t.Fatal("size")
				}
				if phase == "placement" && (strings.Contains(f.Plain(), "HOSTILE WATERS") || !strings.Contains(f.Plain(), "YOUR WATERS")) {
					t.Fatal("placement must show only own waters")
				}
				if phase == "battle" && !strings.Contains(f.Plain(), "HOSTILE WATERS") {
					t.Fatal("missing chart")
				}
				capture(t, f, phase, sd, player)
			}
			for _, e := range []game.Event{{At: game.Coord{X: 9, Y: 9}, Ship: -1}, {At: game.Coord{X: 9, Y: 9}, Hit: true, Ship: -1}, {At: game.Coord{X: 9, Y: 9}, Hit: true, Sunk: true, Ship: 4}} {
				b.Clip = &e
				b.ClipTime = 2
				f := RenderBattle(b)
				r := New(f.W, f.H)
				r.battleClip(b, DefaultState())
				labels := 0
				if e.Sunk {
					labels = 1 // Only the destroyed ship's name accompanies the pixel title.
				}
				if len(r.Texts) != labels || len(CombatControls(b)) != 0 {
					t.Fatal("cutscene has extra labels or invisible mouse controls")
				}
				if strings.Contains(f.Plain(), "J10") || strings.Contains(f.Plain(), "TRANSMISSION") {
					t.Fatal("cutscene contains tactical clutter")
				}
				if e.Ship < 0 && strings.Contains(f.Plain(), "Hive Carrier") {
					t.Fatal("unidentified enemy leaked")
				}
				capture(t, f, "clip-"+map[bool]string{true: "hit", false: "miss"}[e.Hit]+map[bool]string{true: "-sunk"}[e.Sunk], sd, player)
			}
		}
	}
}
func capture(t *testing.T, f *Frame, name string, sd bool, player int) {
	t.Helper()
	dir := os.Getenv("SB2_CAPTURE_DIR")
	if dir == "" {
		return
	}
	size := "hd"
	if sd {
		size = "sd"
	}
	team := "human"
	if player == 1 {
		team = "alien"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(f)
	if err := os.WriteFile(filepath.Join(dir, name+"-"+team+"-"+size+".json"), b, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestReadableShotChartsAndHandoff(t *testing.T) {
	for _, sd := range []bool{true, false} {
		for player := 0; player < 2; player++ {
			v := game.New(player).View(player)
			v.Phase = "battle"
			v.Connected = [2]bool{true, true}
			v.Outgoing[31] = game.Miss
			v.Outgoing[32] = game.Hit
			v.Outgoing[99] = game.Miss
			v.Incoming[40] = game.Miss
			b := BattlePresentation{State: v, SD: sd, Solo: true, Cursor: game.Coord{X: 1, Y: 3}}
			f := RenderBattle(b)
			if !strings.Contains(f.Plain(), "B4 / MISS") {
				t.Fatal("coordinate guide missing")
			}
			grid, _ := CombatBoards(sd, false)
			x, y, cw := grid.X, grid.Y, grid.CW

			accent := Gold
			if player == 1 {
				accent = Green
			}
			if f.Cells[(y/2-1)*f.W+x+cw+cw/2].BG != accent {
				t.Fatal("column not highlighted")
			}
			capture(t, f, "readable-chart", sd, player)
			e := game.Event{Player: 1 - player, At: game.Coord{X: 1, Y: 3}, Ship: -1}
			b.Clip = &e
			b.Handoff = true
			f = RenderBattle(b)
			if strings.Contains(f.Plain(), "ENEMY TAKING AIM") {
				t.Fatal("solo fade contains the old interstitial")
			}
			capture(t, f, "handoff", sd, player)
			b.Handoff = false
			b.ClipTime = .5
			f = RenderBattle(b)
			if strings.Contains(f.Plain(), "ENEMY SHOT / INCOMING") {
				t.Fatal("cutscene contains old shooter banner")
			}
			capture(t, f, "incoming-miss", sd, player)
		}
	}
}
