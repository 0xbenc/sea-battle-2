package play

import (
	tea "charm.land/bubbletea/v2"
	"sea-battle-2/internal/session"
	"strings"
	"testing"
)

func TestStartupFailureCanReturnToMenu(t *testing.T) {
	m, close := New(Options{})
	defer close()
	m.width, m.height = 80, 24
	m.mode = "game"
	m.running = true
	m = apply(m, session.Event{Generation: m.session.Generation(), State: session.Failed, Error: "listen tcp: address already in use"})
	if m.mode != "problem" || m.running || !strings.Contains(m.notice, "using this port") {
		t.Fatalf("failure lost: mode=%s notice=%s", m.mode, m.notice)
	}
	m = apply(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if m.mode != "menu" || m.notice != "" {
		t.Fatal("Back must clear the error and return to menu")
	}
}
