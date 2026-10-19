package play

import (
	tea "charm.land/bubbletea/v2"
	"sea-battle-2/internal/game"
	"sea-battle-2/internal/visual"
)

func (m Model) combatScreen() bool {
	return m.mode == "game" && m.state != nil && (m.state.Phase == "battle" || m.state.Phase == "finished") && !m.invitation && !m.quitting
}
func (m Model) battlePresentation() visual.BattlePresentation {
	v := *m.state
	if m.visible != nil {
		v = *m.visible
	}
	v.Connected = m.state.Connected
	if m.offline {
		v.Connected = [2]bool{}
	}
	b := visual.BattlePresentation{ResultCharts: m.resultCharts, ResultTime: m.t - m.resultStart, Host: m.host, InspectOwn: m.inspectOwn, OwnCursor: m.ownCursor, CombatHelp: m.combatHelp, Hover: m.menuHover, CanUndo: len(m.placementUndo) > 0, PlacementHelp: m.placementHelp, State: v, Fleet: m.fleet, Cursor: m.cursor, Selected: m.selected, Horizontal: m.horizontal, SD: m.sd(), T: m.t, Notice: m.notice, Pending: m.pending, Solo: m.solo, Difficulty: string(m.opts.Difficulty)}
	if len(m.clips) > 0 {
		b.Clip = &m.clips[0].Event
		b.ClipTime = m.clipTime
		if m.handoff() {
			b.Handoff = m.clipTime < handoffDuration
			b.HandoffProgress = m.clipTime / handoffDuration
			b.ClipTime = max(0, m.clipTime-handoffDuration)
		}
	}
	return b
}
func (m Model) combatMouse(v tea.MouseMsg, x, y int) (tea.Model, tea.Cmd) {
	mouse := v.Mouse()
	switch v.(type) {
	case tea.MouseMotionMsg, tea.MouseClickMsg:
	default:
		return m, nil
	}
	_, clicked := v.(tea.MouseClickMsg)
	left := clicked && mouse.Button == tea.MouseLeft
	if m.combatHelp {
		if left {
			m.combatHelp = false
		}
		return m, nil
	}
	b := m.battlePresentation()
	m.menuHover = visual.CombatControlAt(b, x, y)
	if left && m.menuHover != "" {
		code := []rune(m.menuHover)[0]
		switch m.menuHover {
		case "enter":
			code = tea.KeyEnter
		case "space":
			code = ' '
		case "tab":
			code = tea.KeyTab
		case "esc":
			code = tea.KeyEscape
		}
		return m.Update(tea.KeyPressMsg(tea.Key{Code: code}))
	}
	if len(m.clips) > 0 || (b.State.Phase == "finished" && !b.ResultCharts) {
		return m, nil
	}
	for i, c := range visual.CombatFleetControls(m.sd()) {
		if x >= c.X && x < c.X+c.W && y >= c.Y && y < c.Y+c.H {
			m.menuHover = c.Key
			if left && i < len(m.state.Own) && m.state.Own[i].Valid() {
				m.inspectOwn = true
				m.ownCursor = m.state.Own[i].Coord
				m.notice = ""
			}
			return m, nil
		}
	}
	target, own := visual.CombatBoards(m.sd(), m.inspectOwn)
	if cell, ok := own.CellAt(x, y); ok {
		if m.inspectOwn || left {
			m.ownCursor = cell
			m.notice = ""
		}
		if left {
			m.inspectOwn = true
		}
		return m, nil
	}
	if cell, ok := target.CellAt(x, y); ok {
		if m.pending {
			return m, nil
		}
		if m.inspectOwn {
			if left {
				m.inspectOwn = false
				m.cursor = cell
				m.notice = ""
			}
			return m, nil
		}
		if !clicked || left {
			m.cursor = cell
			m.notice = ""
		}
		if left {
			return m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
		}
	}
	return m, nil
}
func moveAim(cursor game.Coord, k string) game.Coord {
	switch k {
	case "up", "w":
		cursor.Y = max(0, cursor.Y-1)
	case "down", "s":
		cursor.Y = min(9, cursor.Y+1)
	case "left", "a":
		cursor.X = max(0, cursor.X-1)
	case "right", "d":
		cursor.X = min(9, cursor.X+1)
	}
	return cursor
}
