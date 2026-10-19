package play

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"sea-battle-2/internal/game"
	"sea-battle-2/internal/peer"
	"sea-battle-2/internal/visual"
	"strings"
	"testing"
)

func TestResultsAfterRealWinningShots(t *testing.T) {
	for _, sd := range []bool{true, false} {
		for player := 0; player < 2; player++ {
			for winner := 0; winner < 2; winner++ {
				for _, hp := range []int{4, 9} {
					m := combatModel(t, sd)
					g := game.New(winner)
					fleets := [2][]game.Placement{game.RandomFleet(41), game.RandomFleet(42)}
					g.SetFleet(0, fleets[0])
					g.SetFleet(1, fleets[1])
					shots := [2][]game.Coord{}
					for shooter := 0; shooter < 2; shooter++ {
						occupied := [100]bool{}
						for i, p := range fleets[1-shooter] {
							for _, c := range game.Cells(p, game.Lengths[i]) {
								shots[shooter] = append(shots[shooter], c)
								occupied[c.Index()] = true
							}
						}
						if shooter != winner {
							shots[shooter] = shots[shooter][:17-hp]
							for i, used := range occupied {
								if !used {
									shots[shooter] = append(shots[shooter], game.Coord{X: i % 10, Y: i / 10})
								}
							}
						}
					}
					update := func() {
						v := g.View(player)
						v.Connected = [2]bool{true, true}
						m = apply(m, peer.Message{Type: "state", View: &v})
					}
					update()
					indices := [2]int{}
					for g.Phase == "battle" {
						p := g.Turn
						if err := g.Fire(p, shots[p][indices[p]], len(g.Events)); err != nil {
							t.Fatal(err)
						}
						indices[p]++
						update()
						if m.visible.Phase == "finished" {
							t.Fatal("result shown before winning cinematic")
						}
						m.t += 1
						m.finishClip()
					}
					b := m.battlePresentation()
					if g.Winner != winner || game.WinnerHullRemaining(b.State) != hp || b.ResultTime != 0 {
						t.Fatal("wrong result or entry clock")
					}
					if !strings.Contains(visual.RenderBattle(b).Plain(), fmt.Sprintf("%d/17 HULL INTACT", hp)) {
						t.Fatal("result missing")
					}
					m = apply(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
					if !m.resultCharts || !strings.Contains(visual.RenderBattle(m.battlePresentation()).Plain(), "HOSTILE WATERS") {
						t.Fatal("cannot inspect final charts")
					}
					m = apply(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
					if m.resultCharts {
						t.Fatal("cannot return to result")
					}
					// Actual centered footer mouse hitbox submits the rematch once.
					c := visual.ResultControls(sd)[0]
					w, h := 160, 48
					if sd {
						w, h = 80, 24
					}
					click := tea.MouseClickMsg(tea.Mouse{X: c.X + 1 + (m.width-w)/2, Y: c.Y + (m.height-h)/2, Button: tea.MouseLeft})
					next, cmd := m.Update(click)
					m = next.(Model)
					if !m.pending || cmd == nil {
						t.Fatal("rematch click not submitted")
					}
					_, cmd = m.Update(click)
					if cmd != nil {
						t.Fatal("duplicate rematch submitted")
					}
					g.RequestRematch(0)
					g.RequestRematch(1)
					update()
					if m.resultCharts || m.state.Phase != "placement" || m.pending {
						t.Fatal("rematch failed to reset result state")
					}
				}
			}
		}
	}
}
