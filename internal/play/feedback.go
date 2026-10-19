package play

import "sea-battle-2/internal/visual"

// A press follows the control key rather than its old bounds, so switching
// display size during a press keeps feedback aligned with the new layout.
func (m Model) feedbackControls() (string, []visual.MenuControl) {
	if m.quitting {
		return "", nil
	}
	switch m.mode {
	case "menu":
		return "menu", visual.MenuControls(m.sd())
	case "faction":
		return "faction", visual.FactionControls(m.sd())
	case "settings":
		return "settings", visual.SettingsControls(m.sd())
	case "help", "credits", "join", "rejoin", "problem":
		return m.mode, visual.FooterControls(m.sd())
	}
	if m.connectingScreen() {
		return "connecting", visual.ConnectingControls(m.sd())
	}
	if m.invitation && m.host {
		return "invitation", visual.InvitationControls(m.sd(), m.invite != "")
	}
	if m.placementScreen() && !m.placementHelp {
		return "placement", visual.PlacementControls(m.sd())
	}
	if m.combatScreen() && !m.combatHelp {
		return "combat", visual.CombatControls(m.battlePresentation())
	}
	return "", nil
}

func (m *Model) markPress(key string) {
	screen, controls := m.feedbackControls()
	if key == "delete" {
		key = "backspace"
	}
	for _, c := range controls {
		if c.Key == key {
			if screen == "combat" && !visual.CombatControlEnabled(m.battlePresentation(), key) {
				return
			}
			m.pressedKey, m.pressedScreen, m.pressedAt = key, screen, m.t
			return
		}
	}
}

func (m Model) drawPress(f *visual.Frame) {
	if m.pressedKey == "" {
		return
	}
	screen, controls := m.feedbackControls()
	if screen != m.pressedScreen {
		return
	}
	for _, c := range controls {
		if c.Key == m.pressedKey {
			visual.PressFeedback(f, c, m.t-m.pressedAt)
		}
	}
}
