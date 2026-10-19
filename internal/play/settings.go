package play

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"sea-battle-2/internal/bot"
)

type preferences struct {
	Compact    bool           `json:"compact"`
	StillMenu  bool           `json:"still_menu"`
	Difficulty bot.Difficulty `json:"difficulty"`
}

func readPreferences(path string) preferences {
	var p preferences
	data, err := os.ReadFile(path)
	if err == nil {
		if json.Unmarshal(data, &p) != nil {
			return preferences{}
		}
	}
	if !p.Difficulty.Valid() {
		p.Difficulty = bot.Medium
	}
	return p
}

func (m *Model) savePreferences() {
	if m.opts.SettingsPath == "" {
		m.notice = "Settings apply for this session."
		return
	}
	p := preferences{m.preferSD, m.stillMenu, m.opts.Difficulty}
	data, err := json.MarshalIndent(p, "", "  ")
	if err == nil {
		err = os.MkdirAll(filepath.Dir(m.opts.SettingsPath), 0700)
	}
	if err == nil {
		var f *os.File
		f, err = os.CreateTemp(filepath.Dir(m.opts.SettingsPath), ".settings-*")
		if err == nil {
			defer os.Remove(f.Name())
			_, err = f.Write(data)
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
			if err == nil {
				err = os.Rename(f.Name(), m.opts.SettingsPath)
			}
		}
	}
	if err != nil {
		m.notice = "Could not save settings; changes apply this session."
	} else {
		m.notice = "Settings saved."
	}
}

func (m Model) settingsLines() []string {
	display, motion := "AUTO (HD when the terminal fits)", "ANIMATED"
	if m.preferSD {
		display = "COMPACT / SD"
	}
	if m.stillMenu {
		motion = "STILL"
	}
	return []string{
		"D  Display: " + display,
		"M  Menu motion: " + motion,
		"", "Default solo opponent: " + strings.ToUpper(string(m.opts.Difficulty)),
		"1 Easy    2 Medium    3 Hard",
		"", "Easy follows hits. Medium hunts. Hard calculates.",
		"Menu motion does not change battle animations.",
		"", m.notice, "", "ESC back    Q quit",
	}
}

var helpLines = []string{
	"Sink all five enemy ships. Players alternate shots.",
	"Chart letters are columns; numbers are rows (B4).",
	"", "Mouse / arrows move    1-5 select ship    X rotate",
	"Click / Enter place    R shuffle    U undo    Del remove",
	"SPACE launch fleet     ENTER fire at the selected cell",
	"", "Single player: choose either navy and a bot difficulty.",
	"Multiplayer: H hosts; send the invitation to your friend.",
	"Your friend uses J to paste it. Keep the host running.",
	"Direct connection: no accounts or relay. Try LAN or IPv6",
	"if router mapping fails, or forward the host's TCP port.",
	"", "ESC / ? back    Q quit",
}
