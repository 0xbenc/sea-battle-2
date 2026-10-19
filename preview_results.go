package main

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"sea-battle-2/internal/game"
	"sea-battle-2/internal/visual"
	"strings"
	"time"
)

type resultPreview struct {
	index, width, height             int
	compact, paused, charts, rematch bool
	elapsed                          float64
	last                             time.Time
}

func (m resultPreview) Init() tea.Cmd { return pulse() }
func (m resultPreview) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case tick:
		now := time.Time(v)
		if !m.paused && !m.last.IsZero() && m.width >= 80 && m.height >= 24 {
			m.elapsed += min(.12, now.Sub(m.last).Seconds())
		}
		m.last = now
		return m, pulse()
	case tea.KeyPressMsg:
		switch k := v.String(); k {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "right", "n":
			m.index = (m.index + 1) % 8
			m.elapsed = 0
			m.rematch = false
		case "left", "p":
			m.index = (m.index + 7) % 8
			m.elapsed = 0
			m.rematch = false
		case "space":
			m.paused = !m.paused
		case "d":
			m.compact = !m.compact
		case "tab":
			m.charts = !m.charts
		case "r":
			m.rematch = !m.rematch
		default:
			if len(k) == 1 && k[0] >= '1' && k[0] <= '8' {
				m.index = int(k[0] - '1')
				m.elapsed = 0
				m.rematch = false
			}
		}
	}
	return m, nil
}

func resultPreviewState(index int) game.View {
	player := index / 4
	win := index%4 < 2
	dominant := index%2 == 1
	winner := player
	if !win {
		winner = 1 - player
	}
	fleets := [2][]game.Placement{game.RandomFleet(71), game.RandomFleet(72)}
	v := game.View{Player: player, Winner: winner, Phase: "finished", Match: 1, Own: fleets[player], Connected: [2]bool{true, true}, Ready: [2]bool{true, true}}
	hp := 4
	if dominant {
		hp = 12
	}
	for target := 0; target < 2; target++ {
		damage := 17
		if target == winner {
			damage = 17 - hp
		}
		marks := &v.Outgoing
		if target == player {
			marks = &v.Incoming
		}
		for ship, p := range fleets[target] {
			length := game.Lengths[ship]
			sunk := damage >= length
			for _, cell := range game.Cells(p, length) {
				if damage == 0 {
					break
				}
				damage--
				mark := game.Hit
				if sunk {
					mark = game.Sunk
				}
				marks[cell.Index()] = mark
				v.Events = append(v.Events, game.Event{Seq: len(v.Events) + 1, Player: 1 - target, At: cell, Hit: true, Ship: -1})
			}
			if sunk {
				last := &v.Events[len(v.Events)-1]
				last.Sunk = true
				last.Ship = ship
			}
		}
	}
	return v
}
func (m resultPreview) View() tea.View {
	if m.width < 80 || m.height < 24 {
		v := tea.NewView("Result preview needs 80 x 24. Resize the terminal. Q exits.")
		v.AltScreen = true
		return v
	}
	sd := m.compact || m.width < 160 || m.height < 48
	state := resultPreviewState(m.index)
	state.Rematch[state.Player] = m.rematch
	f := visual.RenderBattle(visual.BattlePresentation{State: state, SD: sd, ResultTime: m.elapsed, T: m.elapsed, ResultCharts: m.charts, Selected: -1})
	faction := "United States Navy"
	if state.Player == 1 {
		faction = "The Brood"
	}
	outcome := []string{"Close victory", "Overwhelming victory", "Narrow defeat", "Overwhelming defeat"}[m.index%4]
	lines := []string{fmt.Sprintf("PREVIEW %d/8  %s / %s", m.index+1, faction, outcome), "1-8 / arrows: scene   Tab: charts   R: waiting   Space: pause   D: size   Q: exit"}
	for y, line := range lines {
		for x := 0; x < f.W; x++ {
			c := &f.Cells[y*f.W+x]
			c.Ch = " "
			c.BG = visual.Ink
			c.FG = visual.Muted
		}
		for x, r := range []rune(line) {
			if x+2 < f.W {
				f.Cells[y*f.W+x+2].Ch = string(r)
			}
		}
	}
	rows := strings.Split(f.ANSI(), "\n")
	pad := strings.Repeat(" ", max(0, (m.width-f.W)/2))
	for i := range rows {
		rows[i] = pad + rows[i]
	}
	v := tea.NewView(strings.Repeat("\n", max(0, (m.height-f.H)/2)) + strings.Join(rows, "\n"))
	v.AltScreen = true
	v.WindowTitle = "Sea Battle 2 / Result preview"
	return v
}
