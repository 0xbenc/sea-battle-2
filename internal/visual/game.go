package visual

import (
	"sea-battle-2/internal/assets"
	"sea-battle-2/internal/game"
)

// BattlePresentation carries only a player's filtered view, never the host Game.
type BattlePresentation struct {
	ResultCharts                  bool
	ResultTime                    float64
	Host, InspectOwn, CombatHelp  bool
	OwnCursor                     game.Coord
	Hover                         string
	CanUndo, PlacementHelp        bool
	State                         game.View
	Fleet                         []game.Placement
	Cursor                        game.Coord
	Selected                      int
	Horizontal, SD, Pending, Solo bool
	Handoff                       bool
	HandoffProgress               float64
	Difficulty                    string
	T                             float64
	Notice                        string
	Clip                          *game.Event
	ClipTime                      float64
}

func RenderBattle(b BattlePresentation) *Frame {
	if b.Handoff && b.Solo {
		progress := clamp(b.HandoffProgress, 0, 1)
		background := b
		background.Handoff, background.CombatHelp = false, false
		background.ClipTime = 0
		fade := smooth(progress * 2)
		if progress < .5 {
			background.Clip = latestShot(b.State, false)
			if background.Clip != nil {
				background.ClipTime = 2
				if background.Clip.Hit {
					background.ClipTime = 3.5
				}
				if background.Clip.Sunk {
					background.ClipTime = 6
				}
			}
		} else {
			fade = 1 - smooth((progress-.5)*2)
		}
		frame := RenderBattle(background)
		for i := range frame.Cells {
			frame.Cells[i].FG = mix(frame.Cells[i].FG, 0, fade)
			frame.Cells[i].BG = mix(frame.Cells[i].BG, 0, fade)
		}
		return frame
	}
	if b.State.Phase == "placement" && b.Clip == nil && !b.Handoff {
		return renderPlacement(b)
	}
	if b.Clip == nil && !b.Handoff {
		if b.State.Phase == "finished" && !b.ResultCharts && !b.CombatHelp {
			return RenderResult(b)
		}
		return RenderCombat(b)
	}
	w, h := 160, 48
	if b.SD {
		w, h = 80, 24
	}
	var r *Raster
	if b.Handoff {
		background := b
		background.Handoff, background.CombatHelp = false, false
		background.Clip = nil
		r = combatRaster(background)
		r.Texts = nil
		for y := 0; y < r.H; y++ {
			for x := 0; x < r.W; x++ {
				r.blend(x, y, Ink, .6)
			}
		}
		width := min(w-12, 90)
		left, top := (w-width)/2, h/2-3
		r.factionPanel(MenuControl{X: left, Y: top, W: width, H: 6}, Ink, .85)
		r.center(top+1, "ENEMY TAKING AIM", Rust)
		r.center(top+3, "INCOMING FIRE", Bone)
		r.combatHeader(b, "ENEMY TURN", Rust)
	} else {
		r = New(w, h)
		st := DefaultState()
		st.SD = b.SD
		st.T = b.T
		r.battleClip(b, st)

	}
	return r.frame()
}
func (r *Raster) battleClip(b BattlePresentation, st State) {
	e := *b.Clip
	st.T = b.ClipTime
	st.TargetX, st.TargetY = e.At.X, e.At.Y
	// Only a known hull may appear. An ordinary enemy hit is an impact at sea,
	// never a cinematic that accidentally identifies the hidden ship.
	if e.Ship >= 0 {
		s := Scene{Kind: Hit, Ship: (1-e.Player)*5 + e.Ship, Team: e.Player, Duration: 9}
		if e.Sunk {
			s.Kind = Sink
			s.Duration = 13
			st.T *= 2
		}
		r.cinemaScene(st, s, 0, "SIDE", false)
	} else {
		r.ocean(st.T, 0, false)
		if e.Hit {
			r.explosion(r.W/2, r.H/2, st.T, e.Player == 1, 1)
		} else {
			r.splash(r.W/2, r.H/2, st.T, 1)
		}
	}
	word, color := "YOU MISSED!", Cyan
	if e.Player != b.State.Player {
		word = "THEY MISSED!"
	}
	if e.Hit {
		word, color = "YOU HIT!", Gold
		if e.Player != b.State.Player {
			word, color = "YOU WERE HIT!", Rust
		}
	}
	if e.Sunk {
		word, color = "SUNK!", 0xffb466
	}
	row := 4
	if !b.SD {
		row = 8
	}
	// One punchy word; a short white-hot attack settles into the result color.
	color = mix(color, 0xfff5df, 1-smooth(b.ClipTime/.22))
	mask := New(r.W, r.H/2)
	mask.bigWord(word, row, color)
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			if mask.Pixels[y*r.W+x] != 0 {
				r.rect(x-1, y-1, 3, 3, Ink)
			}
		}
	}
	for i, c := range mask.Pixels {
		if c != 0 {
			r.put(i%r.W, i/r.W, c)
		}
	}
	if e.Sunk && e.Ship >= 0 {
		subrow := row + 4
		if !b.SD {
			subrow = row + 7
		}
		r.center(subrow, assets.Names[(1-e.Player)*5+e.Ship], Bone)
	}
}

// frontOcean is scenery only: no title, cinematic HUD, console or modal.
func frontOcean(sd bool, t float64, idle MenuIdle) *Raster {
	w, h, size := 160, 48, "HD"
	if sd {
		w, h, size = 80, 24, "SD"
	}
	r := New(w, h)
	r.ocean(t, 2, false)
	r.menuBackground(idle)
	c := assets.Get("ship-4-side", size)
	x := (w - c.W) / 2
	waterline := r.H * 72 / 100
	y := waterline - c.Anchor.Y
	r.reflection(c, x, waterline+2, t)
	r.wake(x+c.W/2, waterline+1, c.W-5, t, environments[2].crest)
	r.asset(c, x, y, 0, 0, false)
	r.menuForeground(idle)
	return r
}

// RenderMenu composes one title screen directly over the ocean.
func RenderMenu(sd bool, t float64, notice string, state ...MenuIdle) *Frame {
	var idle MenuIdle
	if len(state) > 0 {
		idle = state[0]
	} else {
		idle = menuIdleAt(t)
	}
	return RenderMenuWithHover(sd, t, notice, "", idle)
}

func RenderMenuWithHover(sd bool, t float64, notice, hover string, idle MenuIdle, reactions ...MenuReaction) *Frame {
	r := frontOcean(sd, t, idle)
	logo, subtitle := 5, 12
	if sd {
		logo, subtitle = 3, 7
	}
	r.menuTitle(logo, idle.sheen)
	r.center(subtitle, "OPERATION: UNEXPECTED GUESTS", Gold)
	for _, reaction := range reactions {
		r.menuReaction(reaction, idle)
	}
	r.menuControls(notice, hover)
	const credit = "by TTY.games"
	r.text(r.W-len(credit)-2, r.H/2-1, credit, Muted)
	return r.frame()
}

// RenderFront places secondary menu text on the same ocean, with a soft scrim
// for readability rather than a second screen hidden beneath an opaque panel.
func RenderFront(sd bool, t float64, title string, lines []string, state ...MenuIdle) *Frame {
	var idle MenuIdle
	if len(state) > 0 {
		idle = state[0]
	} else {
		idle = menuIdleAt(t)
	}
	r := frontOcean(sd, t, idle)
	w, h := r.W, r.H/2
	for y := 0; y < r.H; y++ {
		for x := 0; x < w; x++ {
			r.blend(x, y, Ink, .8*(1-smooth(float64(x)/float64(w))))
		}
	}
	width := min(w-8, 112)
	x := (w - width) / 2
	y := 3
	r.text(x+2, y+1, title, Gold)
	row := y + 3
	for _, line := range lines {
		for len(line) > width-4 && row < h-4 {
			r.text(x+2, row, line[:width-4], Bone)
			line = line[width-4:]
			row++
		}
		if row >= h-4 {
			break
		}
		r.text(x+2, row, line, Bone)
		row++
	}
	r.text(2, h-2, "SEA BATTLE 2", Muted)
	return r.frame()
}
