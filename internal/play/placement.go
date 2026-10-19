package play

import (
	"fmt"
	"math/rand/v2"

	tea "charm.land/bubbletea/v2"
	"sea-battle-2/internal/game"
	"sea-battle-2/internal/peer"
	"sea-battle-2/internal/visual"
)

type placementSnapshot struct {
	fleet      []game.Placement
	cursor     game.Coord
	selected   int
	horizontal bool
}

func (m *Model) rememberPlacement() {
	m.placementUndo = append(m.placementUndo, placementSnapshot{append([]game.Placement(nil), m.fleet...), m.cursor, m.selected, m.horizontal})
	if len(m.placementUndo) > 32 {
		m.placementUndo = m.placementUndo[len(m.placementUndo)-32:]
	}
}
func (m Model) placementEditable() bool {
	return m.mode == "game" && m.state != nil && m.state.Phase == "placement" && !m.state.Ready[m.state.Player] && !m.pending && !m.invitation && !m.quitting
}
func (m Model) placementScreen() bool {
	return m.mode == "game" && m.state != nil && m.state.Phase == "placement" && !m.invitation && !m.quitting && len(m.clips) == 0
}
func (m Model) placementDirty() bool {
	return visual.PlacementDirty(visual.BattlePresentation{Fleet: m.fleet, Selected: m.selected, Cursor: m.cursor, Horizontal: m.horizontal})
}
func (m *Model) selectPlacementShip(index int) {
	m.selected = index
	m.notice = ""
	if m.fleet[index].Valid() {
		m.cursor, m.horizontal = m.fleet[index].Coord, m.fleet[index].Horizontal
		return
	}
	// Find the nearest legal preview so auto-advance never starts on the ship
	// just placed. If nothing fits, retain the cursor and show a blocked preview.
	best, nearest := 100, m.cursor
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			p := game.Placement{Coord: game.Coord{X: x, Y: y}, Horizontal: m.horizontal}
			distance := abs(x-m.cursor.X) + abs(y-m.cursor.Y)
			if distance < best && game.CanPlace(m.fleet, index, p) {
				best, nearest = distance, p.Coord
			}
		}
	}
	m.cursor = nearest
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func (m *Model) placementKey(k string) tea.Cmd {
	switch k {
	case "1", "2", "3", "4", "5":
		m.selectPlacementShip(int(k[0] - '1'))
	case "tab":
		m.selectPlacementShip((m.selected + 1) % 5)
	case "x":
		if m.selected >= 0 {
			m.horizontal = !m.horizontal
			m.notice = ""
		}
	case "r":
		m.rememberPlacement()
		m.fleet = game.RandomFleet(rand.Uint64())
		m.selected = -1
		m.notice = ""
	case "u", "ctrl+z":
		if len(m.placementUndo) > 0 {
			old := m.placementUndo[len(m.placementUndo)-1]
			m.placementUndo = m.placementUndo[:len(m.placementUndo)-1]
			m.fleet, m.cursor, m.selected, m.horizontal = old.fleet, old.cursor, old.selected, old.horizontal
			m.notice = ""
		}
	case "backspace", "delete":
		if m.selected >= 0 && m.fleet[m.selected].Valid() {
			m.rememberPlacement()
			m.fleet[m.selected] = game.EmptyFleet()[0]
			m.notice = ""
		}
	case "enter":
		if m.selected < 0 {
			return nil
		}
		p := game.Placement{Coord: m.cursor, Horizontal: m.horizontal}
		if !game.CanPlace(m.fleet, m.selected, p) {
			m.notice = "That position is blocked. Move or rotate the preview."
			return nil
		}
		wasPlaced := m.fleet[m.selected].Valid()
		m.rememberPlacement()
		m.fleet[m.selected] = p
		m.notice = ""
		current := m.selected
		m.selected = -1
		if !wasPlaced {
			for n := 1; n <= 5; n++ {
				i := (current + n) % 5
				if !m.fleet[i].Valid() {
					m.selectPlacementShip(i)
					break
				}
			}
		}
	case "space":
		count := 0
		for _, p := range m.fleet {
			if p.Valid() {
				count++
			}
		}
		if count < 5 {
			m.notice = fmt.Sprintf("Place %d more ships before launching.", 5-count)
			return nil
		}
		if m.placementDirty() {
			m.notice = "Place this move first, or press Esc to cancel it."
			return nil
		}
		if game.ValidateFleet(m.fleet) != nil {
			m.notice = "Check your fleet for overlapping ships."
			return nil
		}
		if m.offline || !m.state.Connected[m.state.Player] {
			m.notice = "Waiting for connection before launch."
			return nil
		}
		m.pending = true
		m.notice = ""
		return m.send(peer.Command{Type: "ready", Fleet: append([]game.Placement(nil), m.fleet...)})
	}
	return nil
}
func (m Model) placementMouse(v tea.MouseMsg, x, y int) (tea.Model, tea.Cmd) {
	mouse := v.Mouse()
	switch v.(type) {
	case tea.MouseMotionMsg, tea.MouseClickMsg, tea.MouseWheelMsg:
	default:
		return m, nil
	}
	_, clicked := v.(tea.MouseClickMsg)
	if m.placementHelp {
		if clicked && mouse.Button == tea.MouseLeft {
			m.placementHelp = false
		}
		return m, nil
	}
	m.menuHover = visual.PlacementControlAt(m.sd(), x, y)
	if clicked && mouse.Button == tea.MouseLeft && m.menuHover != "" {
		key := m.menuHover
		code := []rune(key)[0]
		switch key {
		case "space":
			code = ' '
		case "backspace":
			code = tea.KeyBackspace
		case "esc":
			code = tea.KeyEscape
		}
		return m.Update(tea.KeyPressMsg(tea.Key{Code: code}))
	}
	if !m.placementEditable() {
		return m, nil
	}
	cell, onBoard := visual.PlacementCellAt(m.sd(), x, y)
	if !onBoard {
		return m, nil
	}
	_, wheeled := v.(tea.MouseWheelMsg)
	if (clicked && mouse.Button == tea.MouseRight) || (wheeled && (mouse.Button == tea.MouseWheelUp || mouse.Button == tea.MouseWheelDown)) {
		return m.Update(tea.KeyPressMsg(tea.Key{Code: 'x'}))
	}
	if m.selected >= 0 {
		m.cursor = cell
		m.notice = ""
	}
	if clicked && mouse.Button == tea.MouseLeft {
		if m.selected >= 0 {
			return m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
		}
		for i, p := range m.fleet {
			if !p.Valid() {
				continue
			}
			for _, c := range game.Cells(p, game.Lengths[i]) {
				if c == cell {
					m.selectPlacementShip(i)
					return m, nil
				}
			}
		}
	}
	return m, nil
}
