package play

import (
	tea "charm.land/bubbletea/v2"
	"sea-battle-2/internal/visual"
	"testing"
	"time"
)

func TestMenuMouseTargetsAndCentering(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {160, 48}, {101, 31}, {184, 57}} {
		for _, compact := range []bool{false, true} {
			for _, action := range []struct{ key, mode string }{{"s", "faction"}, {"j", "join"}, {"?", "help"}, {"t", "settings"}} {
				m, close := New(Options{SD: compact})
				m.width, m.height = size[0], size[1]
				w, h := 160, 48
				if m.sd() {
					w, h = 80, 24
				}
				var x, y int
				for _, c := range visual.MenuControls(m.sd()) {
					if c.Key == action.key {
						x, y = c.X+1+(m.width-w)/2, c.Y+(m.height-h)/2
					}
				}
				m = apply(m, tea.MouseMotionMsg(tea.Mouse{X: x, Y: y}))
				if m.menuHover != action.key || m.View().MouseMode != tea.MouseModeAllMotion {
					t.Fatal("hover or tracking missing", size, compact, action.key)
				}
				m = apply(m, tea.MouseClickMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseRight}))
				if m.mode != "menu" {
					t.Fatal("right click activated menu")
				}
				m = apply(m, tea.MouseClickMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseLeft}))
				expectedMouse := tea.MouseModeNone
				if action.mode == "faction" {
					expectedMouse = tea.MouseModeAllMotion
				}
				if m.mode != action.mode || m.View().MouseMode != expectedMouse {
					t.Fatal("click did not navigate or tracking leaked", size, action.key, m.mode)
				}
				close()
			}
		}
	}
}

func TestMenuMouseIgnoresBackgroundAndResize(t *testing.T) {
	m, close := New(Options{})
	defer close()
	m.width, m.height = 80, 24
	m = apply(m, tea.MouseClickMsg(tea.Mouse{X: 40, Y: 10, Button: tea.MouseLeft}))
	if m.mode != "menu" || m.menuHover != "" {
		t.Fatal("ocean activated control")
	}
	m = apply(m, tea.MouseMotionMsg(tea.Mouse{X: 7, Y: 20}))
	if m.menuHover != "s" {
		t.Fatal("missing hover")
	}
	m = apply(m, tea.WindowSizeMsg{Width: 60, Height: 20})
	m = apply(m, tea.MouseClickMsg(tea.Mouse{X: 7, Y: 20, Button: tea.MouseLeft}))
	if m.mode != "menu" || m.menuHover != "" || m.View().MouseMode != tea.MouseModeNone {
		t.Fatal("undersized input enabled")
	}
}

func TestWaterClickCooldown(t *testing.T) {
	m, close := New(Options{})
	defer close()
	m.width, m.height = 80, 24
	click := tea.MouseClickMsg(tea.Mouse{X: 5, Y: 16, Button: tea.MouseLeft})
	m = apply(m, click)
	if m.menuReaction.Kind != "ripple" {
		t.Fatal("water click missing")
	}
	first := m.waterReadyAt
	m.t = 1
	m = apply(m, click)
	if m.waterReadyAt != first || m.reactionStart != 0 {
		t.Fatal("water cooldown retriggered")
	}
	m.waterReadyAt = time.Now().Add(-time.Second)
	m = apply(m, click)
	if m.reactionStart != 1 || !m.waterReadyAt.After(time.Now()) {
		t.Fatal("water did not reset")
	}
}

func TestFactionMousePressAndCancellation(t *testing.T) {
	for _, sd := range []bool{true, false} {
		m, close := New(Options{SD: sd})
		m.width, m.height = 184, 57
		m.mode = "faction"
		w, h := 160, 48
		if sd {
			w, h = 80, 24
		}
		click := func(key string) tea.MouseClickMsg {
			for _, c := range visual.FactionControls(sd) {
				if c.Key == key {
					return tea.MouseClickMsg(tea.Mouse{X: c.X + 1 + (m.width-w)/2, Y: c.Y + (m.height-h)/2, Button: tea.MouseLeft})
				}
			}
			t.Fatal("missing control", key)
			return tea.MouseClickMsg{}
		}
		m = apply(m, click("3"))
		if string(m.opts.Difficulty) != "hard" {
			t.Fatal("difficulty click failed")
		}
		m = apply(m, click("b"))
		if m.mode != "faction" || m.factionPressed != "b" {
			t.Fatal("missing team press feedback")
		}
		stale := factionDeploy{"b", m.factionSerial}
		m = apply(m, click("esc"))
		m = apply(m, stale)
		if m.mode != "menu" {
			t.Fatal("cancelled team click deployed")
		}
		m = apply(m, key("s"))
		m = apply(m, click("a"))
		m = apply(m, factionDeploy{"a", m.factionSerial})
		if m.mode != "connecting" {
			t.Fatal("team click did not deploy")
		}
		close()
	}
}
