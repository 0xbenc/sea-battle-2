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
	m.clips = []shotPlayback{{Event: game.Event{Seq: 1, Ship: -1}}}
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

func TestSoloMenusAndPresentation(t *testing.T) {
	for _, human := range []int{0, 1} {
		m, cleanup := New(Options{})
		m.width, m.height = 80, 24
		m = apply(m, key("s"))
		if m.mode != "faction" {
			t.Fatal("missing faction picker")
		}
		k := "a"
		if human == 1 {
			k = "b"
		}
		m = apply(m, key(k))
		for m.state == nil {
			e, ok := m.session.Next()
			if !ok {
				t.Fatal("closed")
			}
			m = apply(m, e)
		}
		if !m.solo || m.host || m.invitation || m.offline || m.state.Player != human {
			t.Fatal("solo presentation incorrectly configured")
		}
		m = apply(m, key("r"))
		next, cmd := m.Update(key("space"))
		m = next.(Model)
		if cmd == nil {
			t.Fatal("fleet not submitted")
		}
		m = apply(m, cmd())
		e, _ := m.session.Next()
		m = apply(m, e)
		if m.state.Phase != "battle" || m.state.Turn != human {
			t.Fatal("wrong initial turn")
		}
		// A bot response arriving during our shot must queue, not overwrite it.
		next, cmd = m.Update(key("enter"))
		m = next.(Model)
		m = apply(m, cmd())
		e, _ = m.session.Next()
		m = apply(m, e)
		e, _ = m.session.Next()
		m = apply(m, e)
		if len(m.clips) != 2 || m.clips[0].Player != human || m.clips[1].Player == human {
			t.Fatal("cinematic ordering")
		}
		m = apply(m, key("space"))
		m = apply(m, key("space"))
		if len(m.clips) != 1 || !m.handoff() {
			t.Fatal("handoff skipped by repeated input")
		}
		m.clipTime = handoffDuration
		m = apply(m, key("space"))
		if len(m.clips) != 0 || m.state.Turn != human {
			t.Fatal("skip lost turn")
		}
		m = apply(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
		if !m.quitting || !m.returnMenu {
			t.Fatal("no discard confirmation")
		}
		m = apply(m, key("n"))
		if m.quitting || m.returnMenu {
			t.Fatal("cancel left confirmation active")
		}
		cleanup()
	}
}

func TestStoppedSoloDoesNotDismissNewFactionPicker(t *testing.T) {
	m, cleanup := New(Options{})
	defer cleanup()
	m.width, m.height = 80, 24
	m.session.Stop()
	m = apply(m, key("s"))
	m = apply(m, session.Event{Generation: m.session.Generation(), State: session.Stopped})
	if m.mode != "faction" {
		t.Fatal("late stop dismissed new selection")
	}
}

func TestMissTimingAndEnemyHandoff(t *testing.T) {
	for human := 0; human < 2; human++ {
		m, cleanup := New(Options{})
		m.width, m.height = 80, 24
		m.solo = true
		v := game.New(human).View(human)
		m.state = &v
		m.clips = []shotPlayback{{Event: game.Event{Player: human, Ship: -1}}, {Event: game.Event{Player: 1 - human, Ship: -1}}}
		if m.clipDuration() != 2 {
			t.Fatal("miss must last 2 seconds")
		}
		start := time.Now()
		m.last = start
		m.clipTime = 1.95
		m = apply(m, clockTick(start.Add(100*time.Millisecond)))
		if len(m.clips) != 1 || !m.handoff() || m.clipDuration() != 2.8 {
			t.Fatal("missing enemy handoff")
		}
		m.clips[0].Hit = true
		if m.clipDuration() != 4.3 {
			t.Fatal("hit timing changed")
		}
		m.clips[0].Sunk = true
		if m.clipDuration() != 6.8 {
			t.Fatal("sink timing changed")
		}
		cleanup()
	}
}

func TestPlaybackOwnsSnapshotsUntilEachShotFinishes(t *testing.T) {
	m, cleanup := New(Options{})
	defer cleanup()
	m.solo = true
	m.mode = "game"
	m.width, m.height = 80, 24
	g := game.New(0)
	g.SetFleet(0, game.RandomFleet(1))
	g.SetFleet(1, game.RandomFleet(2))
	initial := g.View(0)
	initial.Connected = [2]bool{true, true}
	m = apply(m, peer.Message{Type: "state", View: &initial})
	first := cloneView(initial)
	first.Events = []game.Event{{Seq: 1, Player: 0, At: game.Coord{X: 1, Y: 3}, Ship: -1}}
	first.Outgoing[31] = game.Miss
	first.Turn = 1
	m = apply(m, peer.Message{Type: "state", View: &first})
	second := cloneView(first)
	second.Events = append(second.Events, game.Event{Seq: 2, Player: 1, At: game.Coord{X: 4, Y: 5}, Hit: true, Sunk: true, Ship: 4})
	second.Incoming[54] = game.Sunk
	second.Turn = 0
	second.Phase = "finished"
	second.Winner = 1
	m = apply(m, peer.Message{Type: "state", View: &second})
	if m.state.Phase != "finished" || m.visible.Phase != "battle" || len(m.visible.Events) != 0 || m.visible.Outgoing[31] != game.Unknown || m.visible.Incoming[54] != game.Unknown {
		t.Fatal("authority leaked ahead of playback")
	}
	// Even the quit/menu decision must not treat the unseen winning shot as shown.
	m = apply(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if !m.quitting {
		t.Fatal("unseen finished state skipped leave confirmation")
	}
	m = apply(m, key("n"))
	// Inputs owned by the caller cannot mutate queued presentation snapshots.
	first.Events[0].At = game.Coord{}
	second.Events[0].At = game.Coord{}
	m.finishClip()
	if len(m.visible.Events) != 1 || m.visible.Events[0].At.Index() != 31 || m.visible.Outgoing[31] != game.Miss || m.visible.Incoming[54] != game.Unknown || m.visible.Phase != "battle" || m.visible.Turn != 1 {
		t.Fatal("first shot did not advance to its own snapshot")
	}
	m.clipTime = handoffDuration
	m = apply(m, key("space"))
	if len(m.clips) != 0 || m.visible.Phase != "finished" || m.visible.Winner != 1 || m.visible.Incoming[54] != game.Sunk {
		t.Fatal("winning shot never revealed result")
	}
}

func TestReconnectAndBatchedCatchupReplacePlayback(t *testing.T) {
	for _, offline := range []bool{false, true} {
		m, cleanup := New(Options{})
		m.width, m.height = 80, 24
		v := game.New(0).View(0)
		v.Phase = "battle"
		m = apply(m, peer.Message{Type: "state", View: &v})
		a := cloneView(v)
		a.Events = []game.Event{{Seq: 1, Player: 0, Ship: -1}}
		m = apply(m, peer.Message{Type: "state", View: &a})
		m.offline = offline
		latest := cloneView(a)
		latest.Events = append(latest.Events, game.Event{Seq: 2, Player: 1, Ship: -1}, game.Event{Seq: 3, Player: 0, Ship: -1})
		m = apply(m, peer.Message{Type: "state", View: &latest})
		if len(m.clips) != 0 || len(m.visible.Events) != 3 {
			t.Fatal("catch-up replayed invented intermediate states")
		}
		cleanup()
	}
}

func TestDifficultySelection(t *testing.T) {
	m, cleanup := New(Options{})
	defer cleanup()
	m.width, m.height = 80, 24
	m = apply(m, key("s"))
	for i, d := range []string{"easy", "medium", "hard"} {
		m = apply(m, key(string(rune('1'+i))))
		if string(m.opts.Difficulty) != d || m.mode != "faction" {
			t.Fatal("difficulty selection", d)
		}
	}
}
