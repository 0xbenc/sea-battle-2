package play

import (
	tea "charm.land/bubbletea/v2"
	"reflect"
	"sea-battle-2/internal/visual"
	"testing"
)

func TestInvitationKeyboardAndMousePressFeedback(t *testing.T) {
	for _, sd := range []bool{true, false} {
		m, close := New(Options{SD: sd})
		m.width, m.height = 160, 48
		if sd {
			m.width, m.height = 80, 24
		}
		m.mode, m.host, m.invitation, m.invite = "game", true, true, "test-invitation"
		keyboard := apply(m, key("c"))
		c := visual.InvitationControls(sd, true)[2]
		mouse := apply(m, tea.MouseClickMsg(tea.Mouse{X: c.X + 1, Y: c.Y + 1, Button: tea.MouseLeft}))
		if keyboard.pressedKey != "c" || mouse.pressedKey != "c" {
			t.Fatal("missing press feedback")
		}
		if keyboard.View().Content != mouse.View().Content {
			t.Fatal("mouse and keyboard feedback differ")
		}
		before := keyboard.View().Content
		keyboard.t += .3
		if before == keyboard.View().Content {
			t.Fatal("feedback did not fade")
		}
		close()
	}
}

func TestPressFollowsDisplayResizeAndDoesNotLeakScreens(t *testing.T) {
	m, close := New(Options{})
	defer close()
	m.width, m.height, m.mode = 160, 48, "settings"
	m = apply(m, key("d"))
	if !m.sd() {
		t.Fatal("display toggle did not execute")
	}
	frame := visual.RenderSettings(m.sd(), 0, m.menu.Snapshot(), true, true, "medium", "", "", "dev")
	before := append([]visual.Cell(nil), frame.Cells...)
	m.drawPress(frame)
	if reflect.DeepEqual(before, frame.Cells) {
		t.Fatal("resized switch missing feedback")
	}
	m.mode = "help"
	frame = visual.RenderManual(m.sd(), 0, m.menu.Snapshot(), "")
	before = append([]visual.Cell(nil), frame.Cells...)
	m.drawPress(frame)
	if !reflect.DeepEqual(before, frame.Cells) {
		t.Fatal("press leaked to another screen")
	}
}

func TestUnavailableCombatActionDoesNotFlash(t *testing.T) {
	for _, sd := range []bool{true, false} {
		m := combatModel(t, sd)
		m.pending = true
		m.markPress("enter")
		if m.pressedKey != "" {
			t.Fatal("unavailable fire flashes as enabled")
		}
		m.pending = false
		m.markPress("enter")
		if m.pressedKey != "enter" {
			t.Fatal("available fire lost press feedback")
		}
		m.pressedKey = ""
		m.state.Phase = "finished"
		m.state.Rematch[m.state.Player] = true
		m.markPress("r")
		if m.pressedKey != "" {
			t.Fatal("requested rematch flashes as enabled")
		}
	}
}
