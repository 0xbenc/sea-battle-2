package visual

import (
	"fmt"
	"sea-battle-2/internal/assets"
	"sea-battle-2/internal/game"
	"strings"
)

// BattlePresentation carries only a player's filtered view, never the host Game.
type BattlePresentation struct {
	State                         game.View
	Fleet                         []game.Placement
	Cursor                        game.Coord
	Selected                      int
	Horizontal, SD, Pending, Solo bool
	Handoff                       bool
	Difficulty                    string
	T                             float64
	Notice                        string
	Clip                          *game.Event
	ClipTime                      float64
}

func RenderBattle(b BattlePresentation) *Frame {
	w, h := 160, 48
	if b.SD {
		w, h = 80, 24
	}
	r := New(w, h)
	v := b.State
	st := DefaultState()
	st.SD = b.SD
	st.T = b.T
	accent := Gold
	if v.Player == 1 {
		accent = Green
	}
	r.ocean(b.T, 0, true)
	if b.Handoff {
		alien := v.Player == 0
		color, commander := Gold, "AMERICAN NAVY"
		if alien {
			color, commander = Green, "THE BROOD"
		}
		y := h/2 - 4
		r.console(2, y, w-4, 8, alien)
		r.center(y+1, "ENEMY TURN / "+commander, color)
		r.center(y+3, "INCOMING FIRE", Bone)
		r.center(y+5, "FIRE CONTROL TRANSFERRED", Muted)
	} else if b.Clip != nil {
		r.battleClip(b, st)
	} else {
		cw, ch, x1, x2, py := 6, 4, 12, 92, 18
		if b.SD {
			cw, ch, x1, x2, py = 3, 2, 5, 45, 10
		}
		own := v.Own
		if v.Phase == "placement" && !v.Ready[v.Player] {
			own = b.Fleet
		}
		r.liveBoard(b, x1, py, cw, ch, own, v.Incoming, true)
		r.liveBoard(b, x2, py, cw, ch, nil, v.Outgoing, false)
		r.solidText(x1, py/2-2, "FRIENDLY WATERS", accent, Ink)
		r.solidText(x2, py/2-2, "HOSTILE WATERS", accent, Ink)
		y := h - 9
		if !b.SD {
			y = h - 15
		}
		r.console(2, y, w-4, h-y-3, v.Player == 1)
		gx := 12
		if !b.SD {
			gx = 18
		}
		r.gauge(gx, (y+3)*2, b.T, accent, v.Player == 1)
		r.text(5, y+5, "DIRECTOR", Muted)
		xx := 25
		if !b.SD {
			xx = 45
		}
		line := fmt.Sprintf("TARGET %s  /  SHOT %03d", b.Cursor, len(v.Events)+1)
		status := "WAITING FOR OPPONENT"
		if b.Solo {
			status = "ENEMY DIRECTOR / CALCULATING"
		}
		if v.Phase == "placement" {
			if v.Ready[v.Player] {
				line = "FLEET LOCKED"
				status = "Waiting for the other fleet."
			} else {
				line = fmt.Sprintf("%d / %s [%d]", b.Selected+1, assets.Names[v.Player*5+b.Selected], game.Lengths[b.Selected])
				status = "ENTER place  X rotate  R random  SPACE ready"
			}
		} else if v.Phase == "finished" {
			line = "DEFEAT / THE OCEAN HAS NEW MANAGEMENT"
			if v.Winner == v.Player {
				line = "VICTORY / OCEAN LIBERATED"
			}
			status = "R request rematch"
			if b.Solo {
				status = "R rematch   ESC command menu"
			}
			if v.Rematch[v.Player] {
				status = "Rematch requested. Waiting for opponent."
			}
		} else if v.Turn == v.Player {
			status = "YOUR TURN / ENTER FIRE"
		}
		if !v.Connected[0] || !v.Connected[1] {
			status = "LINK LOST / WAITING FOR BOTH COMMANDERS"
		}
		if b.Pending {
			status = "AWAITING HOST ACKNOWLEDGEMENT"
			if b.Solo {
				status = "COMMAND RECEIVED"
			}
		}
		r.text(xx, y+1, short(line, w-xx-4), Bone)
		r.text(xx, y+3, short(status, w-xx-4), accent)
		if len(v.Events) > 0 {
			e := v.Events[len(v.Events)-1]
			outcome := "MISS"
			if e.Hit {
				outcome = "HIT"
			}
			if e.Sunk {
				outcome = "SUNK"
			}
			shooter := "YOU"
			if e.Player != v.Player {
				shooter = "ENEMY"
			}
			r.text(xx, y+5, fmt.Sprintf("LAST: %s / %s / %s", e.At, outcome, shooter), Bone)
		}
		if !b.SD {
			for i := 0; i < 5; i++ {
				set := "--"
				if len(own) == 5 && own[i].Valid() {
					set = "SET"
					if v.Phase != "placement" {
						remaining := game.Lengths[i]
						for _, c := range game.Cells(own[i], game.Lengths[i]) {
							if v.Incoming[c.Index()] >= game.Hit {
								remaining--
							}
						}
						set = fmt.Sprintf("%d/%d", remaining, game.Lengths[i])
						if remaining == 0 {
							set = "SUNK"
						}
					}
				}
				r.text(7, y+7+i, fmt.Sprintf("%d  %-28s %d  %s", i+1, assets.Names[v.Player*5+i], game.Lengths[i], set), Muted)
			}
			r.text(88, y+7, "Touching ships allowed. One shot per turn.", Muted)
			r.text(88, y+9, "A hit does not grant another shot.", Muted)
		}
	}
	r.plaque(0, 0, w, 2, Ink)
	r.text(2, 0, "SEA BATTLE 2 / OPERATION PEST TIDE", Bone)
	team := "A / AMERICAN NAVY"
	if v.Player == 1 {
		team = "B / THE BROOD"
	}
	if b.Solo {
		team += " / SOLO"
		if b.Difficulty != "" {
			team += " " + strings.ToUpper(b.Difficulty)
		}
	}
	r.text(2, 1, fmt.Sprintf("%s  /  MATCH %d  /  %s", team, v.Match, strings.ToUpper(v.Phase)), accent)
	r.plaque(0, h-2, w, 2, Ink)
	help := "ARROWS/WASD aim  ENTER fire  I invitation  F2 size  Q quit"
	if v.Phase == "placement" {
		help = "ARROWS move  1-5 ship  ENTER place  X rotate  R random  SPACE ready"
	}
	if b.Clip != nil {
		help = "SPACE/ENTER skip transmission   I invitation   Q quit"
	}
	if b.Solo {
		help = strings.ReplaceAll(help, "I invitation", "ESC menu")
		if v.Phase == "placement" {
			help = "ARROWS move 1-5 hull ENTER place X rotate R random SPACE ready ESC menu"
		}
	}
	if b.Handoff {
		help = "ENEMY TAKING AIM...   Q quit"
	}
	r.text(2, h-2, short(help, w-4), Muted)
	notice := b.Notice
	if notice == "" && b.Clip == nil {
		notice = fmt.Sprintf("O miss  X hit  # sunk   |   %s = column %c, row %d", b.Cursor, 'A'+b.Cursor.X, b.Cursor.Y+1)
	}
	r.text(2, h-1, short(notice, w-4), Cyan)
	return r.frame()
}
func (r *Raster) liveBoard(b BattlePresentation, x, y, cw, ch int, fleet []game.Placement, marks [100]game.Mark, own bool) {
	placing := b.State.Phase == "placement" && !b.State.Ready[b.State.Player]
	aiming := (!own && !placing) || (own && placing)
	// Give every square a stable chart background: moving ocean must not obscure marks.
	for row := 0; row < 10; row++ {
		for col := 0; col < 10; col++ {
			bg := RGB(0x0b202a)
			if (row+col)%2 == 1 {
				bg = 0x24424f
			}
			if aiming && (row == b.Cursor.Y || col == b.Cursor.X) {
				bg = 0x254959
			}
			if marks[row*10+col] == game.Miss {
				bg = 0x234957
			}
			for yy := 0; yy < ch; yy++ {
				for xx := 0; xx < cw; xx++ {
					r.put(x+col*cw+xx, y+row*ch+yy, bg)
				}
			}
		}
	}
	for i := 0; i <= 10; i++ {
		r.line(x+i*cw, y, x+i*cw, y+10*ch, 0x41606b)
		r.line(x, y+i*ch, x+10*cw, y+i*ch, 0x41606b)
	}
	for i := 0; i < 10; i++ {
		fg, bg := Bone, Ink
		if aiming && i == b.Cursor.X {
			fg, bg = Ink, Gold
		}
		r.solidText(x+i*cw+cw/2, y/2-1, string(rune('A'+i)), fg, bg)
		fg, bg = Bone, Ink
		if aiming && i == b.Cursor.Y {
			fg, bg = Ink, Gold
		}
		r.solidText(x-3, y/2+i*ch/2+ch/4, fmt.Sprintf("%2d", i+1), fg, bg)
	}
	for i, p := range fleet {
		if i >= 5 || !p.Valid() {
			continue
		}
		size := "board-"
		if p.Horizontal {
			size = "board-horizontal-"
		}
		if b.SD {
			size += "SD"
		} else {
			size += "HD"
		}
		canvas := assets.Get(fmt.Sprintf("ship-%d-top", b.State.Player*5+i), size)
		r.asset(canvas, x+p.X*cw+1, y+p.Y*ch, 0, 0, false)
	}
	if own && placing {
		p := game.Placement{Coord: b.Cursor, Horizontal: b.Horizontal}
		col := Green
		if !game.CanPlace(b.Fleet, b.Selected, p) {
			col = Rust
		}
		for _, c := range game.Cells(p, game.Lengths[b.Selected]) {
			if !c.Valid() {
				continue
			}
			for yy := 0; yy < ch; yy++ {
				for xx := 0; xx < cw; xx++ {
					if (xx+yy)%2 == 0 {
						r.blend(x+c.X*cw+xx, y+c.Y*ch+yy, col, .6)
					}
				}
			}
		}
	}
	for i, m := range marks {
		if m == game.Unknown {
			continue
		}
		symbol, col := "O", Cyan
		if m == game.Hit {
			symbol, col = "X", Rust
		}
		if m == game.Sunk {
			symbol, col = "#", Gold
		}
		r.solidText(x+(i%10)*cw+cw/2, (y+(i/10)*ch)/2+ch/4, symbol, col, Ink)
	}
	if aiming {
		xx, yy := x+b.Cursor.X*cw, y/2+b.Cursor.Y*ch/2+ch/4
		// Brackets belong inside the selected cell, never on a shared cell boundary.
		r.solidText(xx, yy, "[", Gold, Ink)
		r.solidText(xx+cw-1, yy, "]", Gold, Ink)
	}

}
func (r *Raster) battleClip(b BattlePresentation, st State) {
	e := *b.Clip
	st.T = b.ClipTime
	st.TargetX, st.TargetY = e.At.X, e.At.Y
	label := "MISS / THE WATER HAS LEARNED ITS LESSON"
	if e.Hit {
		label = "HIT / CONTACT CONFIRMED"
	}
	if e.Sunk {
		label = "SUNK / " + assets.Names[(1-e.Player)*5+e.Ship]
	}
	// Only a known hull may appear. An ordinary enemy hit is an impact at sea,
	// never a cinematic that accidentally identifies the hidden ship.
	if e.Ship >= 0 {
		s := Scene{Kind: Hit, Ship: (1-e.Player)*5 + e.Ship, Team: e.Player, Duration: 9}
		if e.Sunk {
			s.Kind = Sink
			s.Duration = 13
			st.T *= 2
		}
		r.cinema(st, s, 0, "SIDE")
	} else {
		r.ocean(st.T, 0, false)
		if e.Hit {
			r.explosion(r.W/2, r.H/2, st.T, e.Player == 1, 1)
		} else {
			r.splash(r.W/2, r.H/2, st.T, 1)
		}
	}
	shooter, color := "YOUR SHOT / OUTGOING", Gold
	if e.Player != b.State.Player {
		shooter, color = "ENEMY SHOT / INCOMING", Rust
	}
	r.plaque(2, 3, r.W-4, 3, Ink)
	r.center(4, shooter, color)
	r.plaque(2, r.H/2-7, r.W-4, 3, Ink)
	r.center(r.H/2-6, short(fmt.Sprintf("%s  /  %s", e.At, label), r.W-6), Gold)
}

// RenderFront reuses the opening art beneath a readable connection console.
func RenderFront(sd bool, t float64, title string, lines []string) *Frame {
	w, h := 160, 48
	if sd {
		w, h = 80, 24
	}
	r := New(w, h)
	st := DefaultState()
	st.SD = sd
	st.T = t
	r.cinema(st, Scenes[0], 2, "SIDE")
	// The scene supplies the ocean and ships, not captions behind the menu.
	r.Texts = nil
	width := min(w-8, 112)
	x := (w - width) / 2
	y := 3
	r.plaque(x, y, width, h-6, Ink)
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
	r.solidText(2, h-2, "SEA BATTLE 2 / NO ACCOUNTS / NO SHARED SERVICES", Muted, Ink)
	return r.frame()
}
