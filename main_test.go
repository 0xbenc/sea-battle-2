package main

import (
	tea "charm.land/bubbletea/v2"
	"sea-battle-2/internal/visual"
	"testing"
	"time"
)

func update(m model, msg tea.Msg) model { next, _ := m.Update(msg); return next.(model) }
func key(m model, s string) model {
	return update(m, tea.KeyPressMsg(tea.Key{Code: []rune(s)[0], Text: s}))
}
func TestResizeAndPauseClock(t *testing.T) {
	m := model{state: visual.DefaultState()}
	m = update(m, tea.WindowSizeMsg{Width: 160, Height: 48})
	if m.state.SD {
		t.Fatal("HD not selected")
	}
	m = update(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if !m.state.SD {
		t.Fatal("no fallback")
	}
	m = update(m, tea.WindowSizeMsg{Width: 160, Height: 48})
	if m.state.SD {
		t.Fatal("HD preference lost")
	}
	now := time.Now()
	m = update(m, tick(now))
	m = update(m, tick(now.Add(40*time.Millisecond)))
	if m.state.T <= 0 {
		t.Fatal("clock not running")
	}
	m.state.Paused = true
	before := m.state.T
	m = update(m, tick(now.Add(80*time.Millisecond)))
	if m.state.T != before {
		t.Fatal("paused clock advanced")
	}
	m.state.Paused = false
	m = update(m, tea.WindowSizeMsg{Width: 70, Height: 20})
	m = update(m, tick(now.Add(time.Second)))
	m = update(m, tick(now.Add(2*time.Second)))
	if m.state.T != before {
		t.Fatal("undersize advanced clock")
	}
}
func TestFocusAndTour(t *testing.T) {
	m := model{state: visual.DefaultState(), width: 160, height: 48}
	m.state.T = visual.Scenes[0].Duration - .01
	now := time.Now()
	m = update(m, tick(now))
	m = update(m, tick(now.Add(50*time.Millisecond)))
	if m.state.Scene != 1 || m.state.T != 0 {
		t.Fatal("tour did not advance")
	}
	m = update(m, tea.BlurMsg{})
	m = update(m, tick(now.Add(time.Second)))
	m = update(m, tick(now.Add(2*time.Second)))
	if m.state.T != 0 {
		t.Fatal("blur advanced playback")
	}
	m = update(m, tea.FocusMsg{})
	m = update(m, tick(now.Add(3*time.Second)))
	if m.state.T != 0 {
		t.Fatal("focus restoration jumped clock")
	}
}
func TestAimLimitsAndWeather(t *testing.T) {
	m := model{state: visual.DefaultState()}
	m.state.Scene = visual.Find("human-station")
	for i := 0; i < 30; i++ {
		m = key(m, "a")
		m = key(m, "w")
	}
	if m.state.TargetX != 0 || m.state.TargetY != 0 {
		t.Fatal("aim lower bound")
	}
	for i := 0; i < 30; i++ {
		m = key(m, "d")
		m = key(m, "s")
	}
	if m.state.TargetX != 9 || m.state.TargetY != 9 {
		t.Fatal("aim upper bound")
	}
	m.change(visual.Find("hive"))
	m = key(m, "w")
	if m.state.Weather != 0 {
		t.Fatal("weather cycle should advance from scene default")
	}
}

func TestFirePlaysWholeSequence(t *testing.T) {
	m := model{state: visual.DefaultState()}
	m.state.Scene = visual.Find("human-station")
	m.state.Paused = true
	m.state.TargetX = 2
	m.state.TargetY = 1
	m = update(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if m.state.Scene != visual.Find("exchange") || !m.state.ShotMiss || m.state.Paused || m.state.Tour {
		t.Fatal("fire did not start deterministic full sequence")
	}
}

func TestFinalSequencePlotAcceptsNextShot(t *testing.T) {
	m := model{state: visual.DefaultState()}
	m.state.Scene = visual.Find("counterexchange")
	m.state.T = 14
	m = key(m, "d")
	if m.state.TargetX != 7 {
		t.Fatal("final plot does not accept aiming")
	}
	m = update(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if m.state.Scene != visual.Find("counterexchange") || m.state.T != 0 {
		t.Fatal("final plot does not accept fire")
	}
}
