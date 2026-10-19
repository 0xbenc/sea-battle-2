package play

import (
	tea "charm.land/bubbletea/v2"
	"reflect"
	"sea-battle-2/internal/game"
	"sea-battle-2/internal/peer"
	"sea-battle-2/internal/visual"
	"testing"
)

func placementModel(t *testing.T, sd bool) Model {
	t.Helper()
	m, close := New(Options{SD: sd})
	t.Cleanup(close)
	m.width, m.height = 184, 57
	m.mode = "game"
	m.solo = true
	v := game.New(0).View(0)
	v.Connected = [2]bool{true, true}
	m.state = &v
	return m
}
func placementClick(m Model, cell game.Coord, button tea.MouseButton) tea.MouseClickMsg {
	l := visual.PlacementGeometry(m.sd())
	w, h := 160, 48
	if m.sd() {
		w, h = 80, 24
	}
	return tea.MouseClickMsg(tea.Mouse{X: l.X + cell.X*l.CW + l.CW/2 + (m.width-w)/2, Y: (l.Y+cell.Y*l.CH+l.CH/2)/2 + (m.height-h)/2, Button: button})
}
func TestPlacementMouseBuildEditUndo(t *testing.T) {
	for _, sd := range []bool{true, false} {
		m := placementModel(t, sd)
		for n := 0; n < 5; n++ {
			selected := m.selected
			if selected < 0 {
				t.Fatal("auto advance stopped early")
			}
			p := game.Placement{Coord: m.cursor, Horizontal: m.horizontal}
			if !game.CanPlace(m.fleet, selected, p) {
				t.Fatal("auto advance chose blocked preview")
			}
			click := placementClick(m, m.cursor, tea.MouseLeft)
			m = apply(m, click)
			after := m.cursor
			m = apply(m, tea.MouseReleaseMsg(click))
			if m.cursor != after {
				t.Fatal("release moved the next preview")
			}
			if m.fleet[selected] != p {
				t.Fatal("click did not place hull")
			}
		}
		if game.ValidateFleet(m.fleet) != nil || m.selected != -1 {
			t.Fatal("fleet not ready for review")
		}
		fleet := append([]game.Placement(nil), m.fleet...)
		m = apply(m, placementClick(m, m.fleet[2].Coord, tea.MouseLeft))
		if m.selected != 2 || m.cursor != m.fleet[2].Coord || m.horizontal != m.fleet[2].Horizontal {
			t.Fatal("click edit lost original pose")
		}
		m = apply(m, placementClick(m, m.cursor, tea.MouseRight))
		if m.horizontal == m.fleet[2].Horizontal {
			t.Fatal("right click did not rotate")
		}
		m = apply(m, key("space"))
		if m.pending {
			t.Fatal("uncommitted move launched")
		}
		m = apply(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
		if m.selected != -1 || !reflect.DeepEqual(m.fleet, fleet) || m.quitting {
			t.Fatal("cancel move changed fleet or left game")
		}
		m = apply(m, key("3"))
		m = apply(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
		if m.fleet[2].Valid() {
			t.Fatal("remove failed")
		}
		m = apply(m, key("u"))
		if !reflect.DeepEqual(m.fleet, fleet) {
			t.Fatal("undo removal failed")
		}
		m = apply(m, key("r"))
		m = apply(m, key("u"))
		if !reflect.DeepEqual(m.fleet, fleet) {
			t.Fatal("undo shuffle failed")
		}
	}
}
func TestPlacementInvalidAndLockedInput(t *testing.T) {
	m := placementModel(t, true)
	m.selected = 4
	m.cursor = game.Coord{X: 9, Y: 9}
	m = apply(m, key("enter"))
	if m.fleet[4].Valid() || len(m.placementUndo) != 0 {
		t.Fatal("invalid placement mutated history")
	}
	m = apply(m, key("space"))
	if m.pending {
		t.Fatal("incomplete fleet launched")
	}
	m = apply(m, key("r"))
	fleet := append([]game.Placement(nil), m.fleet...)
	m.pending = true
	m = apply(m, key("r"))
	m = apply(m, key("u"))
	if !reflect.DeepEqual(m.fleet, fleet) {
		t.Fatal("pending fleet mutated")
	}
	m.pending = false
	m.state.Ready[0] = true
	m = apply(m, key("r"))
	m = apply(m, placementClick(m, game.Coord{}, tea.MouseLeft))
	if !reflect.DeepEqual(m.fleet, fleet) {
		t.Fatal("locked fleet mutated")
	}
}

func TestPlacementLaunchResetsCombatAim(t *testing.T) {
	m := placementModel(t, true)
	m.cursor = game.Coord{X: 9, Y: 8}
	m.placementHelp = true
	m.rememberPlacement()
	next := *m.state
	next.Phase = "battle"
	next.Ready = [2]bool{true, true}
	m = apply(m, peer.Message{Type: "state", View: &next})
	if m.cursor != (game.Coord{}) || m.placementHelp || len(m.placementUndo) != 0 {
		t.Fatal("deployment state leaked into combat")
	}
}
