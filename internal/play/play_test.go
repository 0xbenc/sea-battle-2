package play

import (
	tea "charm.land/bubbletea/v2"
	"errors"
	"sea-battle-2/internal/game"
	"sea-battle-2/internal/peer"
	"sea-battle-2/internal/session"
	"strings"
	"testing"
	"time"
)

func key(k string) tea.KeyPressMsg {
	code := rune(0)
	switch k {
	case "enter":
		code = tea.KeyEnter
	case "space":
		code = ' '
	case "right":
		code = tea.KeyRight
	default:
		code = []rune(k)[0]
	}
	return tea.KeyPressMsg(tea.Key{Code: code})
}
func apply(m Model, msg tea.Msg) Model { next, _ := m.Update(msg); return next.(Model) }
func TestPlacementAndInputGates(t *testing.T) {
	m, close := New(Options{})
	defer close()
	m.width, m.height = 80, 24
	m.mode = "game"
	v := game.New(0).View(0)
	v.Connected = [2]bool{true, true}
	m.state = &v
	m = apply(m, key("r"))
	if err := game.ValidateFleet(m.fleet); err != nil {
		t.Fatal(err)
	}
	m = apply(m, key("5"))
	if m.selected != 4 {
		t.Fatal("ship selector")
	}
	m = apply(m, key("x"))
	if !m.horizontal {
		t.Fatal("rotation")
	}
	m.width = 70
	m = apply(m, key("right"))
	if m.cursor.X != 0 {
		t.Fatal("undersize accepted aim")
	}
	m.width = 80
	m = apply(m, key("right"))
	if m.cursor.X != 1 {
		t.Fatal("aim")
	}
	m.offline = true
	m = apply(m, key("space"))
	if m.pending {
		t.Fatal("submitted offline fleet")
	}
	m.state.Phase = "battle"
	m.state.Turn = 1
	m.offline = false
	m = apply(m, key("enter"))
	if m.pending || !strings.Contains(m.notice, "turn") {
		t.Fatal("accepted enemy turn")
	}
}
func TestLiveEventsAndResync(t *testing.T) {
	m, close := New(Options{})
	defer close()
	m.width, m.height = 80, 24
	m.mode = "game"
	old := game.New(0).View(0)
	m.state = &old
	next := old
	next.Events = []game.Event{{Seq: 1, At: game.Coord{X: 3, Y: 2}, Ship: -1}}
	m = apply(m, peer.Message{Type: "state", View: &next})
	if len(m.clips) != 1 {
		t.Fatal("live result not queued")
	}
	m = apply(m, key("space"))
	if len(m.clips) != 0 {
		t.Fatal("skip")
	}
	m.offline = true
	next.Events = append(next.Events, game.Event{Seq: 2, Ship: -1})
	m = apply(m, peer.Message{Type: "state", View: &next})
	if len(m.clips) != 0 {
		t.Fatal("replayed historical shots")
	}
	m = apply(m, peer.Message{Type: "offline"})
	if !m.offline {
		t.Fatal("offline gate")
	}
}
func TestLivePlaybackContinuesWithoutFocus(t *testing.T) {
	m, cleanup := New(Options{})
	defer cleanup()
	m.width, m.height = 80, 24
	start := time.Now()
	m.last = start
	m.clips = []game.Event{{Seq: 1, Ship: -1}}
	m.clipTime = 3.45
	m = apply(m, tea.BlurMsg{})
	m = apply(m, clockTick(start.Add(100*time.Millisecond)))
	if m.t == 0 || len(m.clips) != 0 {
		t.Fatal("unfocused terminal did not advance and finish its cinematic")
	}
	m = apply(m, tea.FocusMsg{})
	m = apply(m, clockTick(start.Add(200*time.Millisecond)))
	if m.t < .19 {
		t.Fatal("focus regain interrupted playback")
	}
	before := m.t
	m.width = 70
	m = apply(m, clockTick(start.Add(300*time.Millisecond)))
	if m.t != before {
		t.Fatal("undersized terminal advanced playback")
	}
}
func TestSanitizeInvitationInput(t *testing.T) {
	if got := cleanInput("sb2:ab\ncd\x1b\u202ee"); got != "sb2:abcde" {
		t.Fatal(got)
	}
}

func TestGuestArrivalOpensHostDeployment(t *testing.T) {
	m, cleanup := New(Options{})
	defer cleanup()
	m.mode = "game"
	m.invitation = true
	waiting := game.New(0).View(0)
	waiting.Connected[0] = true
	m = apply(m, peer.Message{Type: "state", View: &waiting})
	if !m.invitation {
		t.Fatal("invitation closed before the guest arrived")
	}
	joined := waiting
	joined.Connected[1] = true
	m = apply(m, peer.Message{Type: "state", View: &joined})
	if m.invitation {
		t.Fatal("host still needs Enter after guest joins")
	}
	// A later state update must not close a deliberately reopened invitation.
	m.invitation = true
	update := joined
	update.Ready[1] = true
	m = apply(m, peer.Message{Type: "state", View: &update})
	if !m.invitation {
		t.Fatal("ordinary update closed reopened invitation")
	}
}

func TestStaleSessionResultsCannotReplaceCurrentPresentation(t *testing.T) {
	m, cleanup := New(Options{})
	defer cleanup()
	m.session.Stop()
	current := m.session.Generation()
	m.mode = "game"
	m.pending = true
	m.notice = "current"
	m.invite = "current invitation"
	m = apply(m, session.Event{Generation: current - 1, State: session.Failed, Error: "obsolete failure", Invitation: "obsolete invitation"})
	m = apply(m, sent{generation: current - 1, err: errors.New("obsolete write")})
	if m.mode != "game" || !m.pending || m.notice != "current" || m.invite != "current invitation" {
		t.Fatal("stale result altered active UI")
	}
}
func TestServerRejectionSurvivesStateResync(t *testing.T) {
	m, cleanup := New(Options{})
	defer cleanup()
	v := game.New(0).View(0)
	m.state = &v
	m.pending = true
	m = apply(m, peer.Message{Type: "error", Error: "wait for your turn"})
	m = apply(m, peer.Message{Type: "state", View: &v})
	if m.notice != "wait for your turn" {
		t.Fatal("resync erased rejection")
	}
}
func TestRepeatedEnterDoesNotQueueMultipleShots(t *testing.T) {
	m, cleanup := New(Options{})
	defer cleanup()
	m.mode = "game"
	m.width, m.height = 80, 24
	g := game.New(0)
	g.SetFleet(0, game.RandomFleet(1))
	g.SetFleet(1, game.RandomFleet(2))
	v := g.View(0)
	v.Connected = [2]bool{true, true}
	m.state = &v
	next, command := m.Update(key("enter"))
	m = next.(Model)
	if command == nil || !m.pending {
		t.Fatal("first shot not submitted")
	}
	for i := 0; i < 50; i++ {
		next, command = m.Update(key("enter"))
		m = next.(Model)
		if command != nil {
			t.Fatal("repeated input queued another command")
		}
	}
}
