package play

import (
	tea "charm.land/bubbletea/v2"
	"sea-battle-2/internal/game"
	"sea-battle-2/internal/visual"
	"testing"
)

func combatModel(t *testing.T, sd bool) Model {
	m := placementModel(t, sd)
	m.state.Phase = "battle"
	m.state.Turn = m.state.Player
	m.state.Own = game.RandomFleet(8)
	return m
}
func combatMouseAt(m Model, own bool, c game.Coord) tea.MouseClickMsg {
	target, friendly := visual.CombatBoards(m.sd(), m.inspectOwn)
	board := target
	if own {
		board = friendly
	}
	w, h := 160, 48
	if m.sd() {
		w, h = 80, 24
	}
	return tea.MouseClickMsg(tea.Mouse{X: board.X + c.X*board.CW + board.CW/2 + (m.width-w)/2, Y: (board.Y+c.Y*board.CH+board.CH/2)/2 + (m.height-h)/2, Button: tea.MouseLeft})
}
func TestCombatMouseAimAndFireGates(t *testing.T) {
	for _, sd := range []bool{true, false} {
		for _, blocked := range []string{"", "enemy", "pending", "miss", "offline", "cinema"} {
			m := combatModel(t, sd)
			cell := game.Coord{X: 1, Y: 3}
			switch blocked {
			case "enemy":
				m.state.Turn = 1 - m.state.Player
			case "pending":
				m.pending = true
			case "miss":
				m.state.Outgoing[cell.Index()] = game.Miss
			case "offline":
				m.offline = true
			case "cinema":
				m.clips = []shotPlayback{{Event: game.Event{Player: m.state.Player, At: game.Coord{}, Ship: -1}}}
			}
			click := combatMouseAt(m, false, cell)
			m = apply(m, tea.MouseMotionMsg(click))
			m = apply(m, click)
			m = apply(m, tea.MouseReleaseMsg(click))
			if blocked == "" {
				if !m.pending || m.cursor != cell {
					t.Fatal("click did not submit aimed shot", sd)
				}
				if m.state.Outgoing[cell.Index()] != game.Unknown {
					t.Fatal("unacknowledged hit appeared")
				}
			} else if blocked != "pending" && m.pending {
				t.Fatal("blocked click fired", blocked)
			}
		}
	}
}
func TestCombatInspectionDoesNotFireOrMoveAim(t *testing.T) {
	for _, sd := range []bool{true, false} {
		m := combatModel(t, sd)
		m.cursor = game.Coord{X: 7, Y: 8}
		aim := m.cursor
		m = apply(m, combatMouseAt(m, true, game.Coord{X: 2, Y: 3}))
		if !m.inspectOwn || m.cursor != aim || m.pending {
			t.Fatal("small friendly chart fired or moved aim")
		}
		m = apply(m, combatMouseAt(m, true, game.Coord{X: 5, Y: 5}))
		if m.ownCursor != (game.Coord{X: 5, Y: 5}) || m.cursor != aim || m.pending {
			t.Fatal("friendly inspection changed targeting")
		}
		m = apply(m, key("enter"))
		if m.inspectOwn || m.pending {
			t.Fatal("enter from inspection must return without firing")
		}
		m = apply(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
		m = apply(m, combatMouseAt(m, false, game.Coord{X: 1, Y: 1}))
		if m.inspectOwn || m.pending || m.cursor != (game.Coord{X: 1, Y: 1}) {
			t.Fatal("small hostile chart should expand without firing")
		}
		m = apply(m, key("?"))
		m = apply(m, combatMouseAt(m, false, game.Coord{X: 1, Y: 1}))
		if m.combatHelp || m.pending {
			t.Fatal("closing help fired a shot")
		}
	}
}

func TestCinematicSkipMouseMatchesKeyboard(t *testing.T) {
	for _, sd := range []bool{true, false} {
		m := placementModel(t, sd)
		m.state.Phase = "battle"
		after := cloneView(*m.state)
		m.clips = []shotPlayback{{Event: game.Event{Player: m.state.Player, At: game.Coord{}, Ship: -1}, After: after}}
		m.clipTime = .5
		c := visual.CombatControls(m.battlePresentation())[0]
		w, h := 160, 48
		if m.sd() {
			w, h = 80, 24
		}
		m = apply(m, tea.MouseClickMsg(tea.Mouse{X: c.X + (m.width-w)/2, Y: c.Y + (m.height-h)/2, Button: tea.MouseLeft}))
		if len(m.clips) != 0 {
			t.Fatal("clicking Skip did not finish cinematic")
		}
	}
}
