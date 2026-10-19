package play

import (
	"bytes"
	"github.com/charmbracelet/x/ansi"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"sea-battle-2/internal/bot"
)

func TestMenuHelpAndSavedSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "settings.json")
	m, close := New(Options{SettingsPath: path})
	defer close()
	m.width, m.height = 160, 48
	m = apply(m, key("?"))
	if m.mode != "help" || !strings.Contains(ansi.Strip(m.View().Content), "FIELD MANUAL") {
		t.Fatal("help did not open")
	}
	m = apply(m, key("h"))
	if m.mode != "help" || m.running {
		t.Fatal("help leaked host shortcut")
	}
	m = apply(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	m = apply(m, key("t"))
	if m.mode != "settings" {
		t.Fatal("settings did not open")
	}
	for _, k := range []string{"d", "m", "3"} {
		m = apply(m, key(k))
	}
	if !m.sd() || !m.stillMenu || m.opts.Difficulty != bot.Hard {
		t.Fatal("settings did not apply")
	}
	m.last = time.Now()
	before := m.menu.Snapshot()
	m = apply(m, clockTick(m.last.Add(time.Second/24)))
	if m.menuTime != 0 || m.menu.Snapshot() != before {
		t.Fatal("still menu kept animating")
	}
	restored, cleanup := New(Options{SettingsPath: path})
	defer cleanup()
	if !restored.preferSD || !restored.stillMenu || restored.opts.Difficulty != bot.Hard {
		t.Fatal("settings did not survive reload")
	}
	override, done := New(Options{SettingsPath: path, DisplayExplicit: true, Difficulty: bot.Easy})
	defer done()
	if override.preferSD || override.opts.Difficulty != bot.Easy {
		t.Fatal("settings overrode explicit command line options")
	}
}

// This fixture is the format shipped in v1.0.0. Loading a newer game must not
// rewrite preferences or reset player choices; saving must remain readable.
func TestReleaseSettingsCompatibility(t *testing.T) {
	original, err := os.ReadFile("testdata/settings-v1.0.0.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	m, cleanup := New(Options{SettingsPath: path, Version: "1.0.1"})
	defer cleanup()
	if !m.preferSD || !m.stillMenu || m.opts.Difficulty != bot.Hard {
		t.Fatal("release upgrade lost preferences")
	}
	loaded, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(original, loaded) {
		t.Fatal("loading rewrote preferences")
	}
	m.savePreferences()
	if got := readPreferences(path); !got.Compact || !got.StillMenu || got.Difficulty != bot.Hard {
		t.Fatal("saved preferences no longer compatible")
	}
}

func TestSettingsShowsBuildVersion(t *testing.T) {
	for _, width := range []int{80, 160} {
		m, cleanup := New(Options{Version: "1.0.1"})
		m.width, m.height = width, width*3/10
		m.mode = "settings"
		if !strings.Contains(ansi.Strip(m.View().Content), "Version 1.0.1") {
			t.Fatalf("version missing at width %d", width)
		}
		cleanup()
	}
}
