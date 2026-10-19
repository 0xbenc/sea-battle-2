package play

import (
	"encoding/json"
	"os"
	"path/filepath"

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
