package visual

import (
	"fmt"
	"sea-battle-2/internal/assets"
)

func (r *Raster) tactical(st State, s Scene, weather int) {
	r.ocean(st.T, weather, true)
	hd := !st.SD
	cw, ch := 3, 2
	x1, x2, py := 5, 45, 10
	if hd {
		cw, ch = 6, 4
		x1, x2, py = 12, 92, 18
	}
	accent := Gold

	if s.Team == 1 {
		accent = Green

	}
	row := py/2 - 2
	r.solidText(x1, row, "FRIENDLY WATERS", accent, Ink)
	r.solidText(x2, row, "HOSTILE WATERS", accent, Ink)
	r.board(st, x1, py, cw, ch, true, s.Team)
	r.board(st, x2, py, cw, ch, false, s.Team)
	if hd {
		r.text(4, 3, "TACTICAL PLOT  /  LIVE OPTICAL SURFACE", Muted)
		r.text(4, 4, "Ships concealed on hostile grid. Only confirmed shots are shown.", Bone)
		fleetStatus := "FLEET  5/5    HULLS  17/17"
		if s.ID == "deployment" {
			fleetStatus = "4 SET / 1 PLACEMENT GHOST"
		}
		r.text(x1, py/2+ch*5+2, fleetStatus, Muted)
		r.text(x2, py/2+ch*5+2, fmt.Sprintf("CONTACT  %c%d    SOLUTION READY", 'A'+st.TargetX, st.TargetY+1), Muted)
	}
	y := r.H/2 - 11
	if !hd {
		y = r.H/2 - 8
	}
	height := 6
	if hd {
		height = 9
	}
	r.console(2, y, r.W-4, height, s.Team == 1)
	if s.Team == 0 {
		for _, xx := range []int{3, r.W - 4} {
			r.text(xx, y, "⊕", Muted)
			r.text(xx, y+6, "⊕", Muted)
		}
		r.text(6, y, " MK.09 / GUN DIRECTOR ", Gold)
		gy := y + 3
		if hd {
			gy = y + 5
		}
		r.gauge(13, gy*2, st.T, Gold, false)
		if hd {
			r.gauge(39, (y+5)*2, st.T+1.5, Cyan, false)
			r.text(6, y+8, "PRESSURE / PSI", Muted)
			r.text(32, y+8, "TRAIN / DEG", Muted)
		}
	} else {
		for x := 3; x < r.W-4; x += 6 {
			r.line(x, y*2, x+2, y*2-2, 0x926e7c)
			r.line(x+2, y*2-2, x+4, y*2, 0x443243)
		}
		r.text(6, y, " BROOD / NEURAL DIRECTOR ", Green)
		gy := y + 3
		if hd {
			gy = y + 5
		}
		r.gauge(13, gy*2, st.T, Green, true)
		if hd {
			r.gauge(39, (y+5)*2, st.T+2, Green, true)
			r.text(6, y+8, "VENOM / SAT", Muted)
			r.text(32, y+8, "SYNAPSE / Hz", Muted)
		}
	}
	xx := 25
	if hd {
		xx = 60
	}
	r.text(xx, y+2, fmt.Sprintf("TARGET  %c%d", 'A'+st.TargetX, st.TargetY+1), Bone)
	status := "● SOLUTION LOCKED"
	if st.Confirmed {
		status = "× HIT CONFIRMED"
		if st.ShotMiss {
			status = "· MISS CONFIRMED"
		}
	}
	r.text(xx, y+3, status, accent)
	action := "WASD AIM / ENTER FIRE"
	if s.ID == "deployment" {
		action = "WASD POSITION / X ROTATE"
		status := "● VALID PLACEMENT"
		col := accent
		if !placementLegal(st.TargetX, st.TargetY, st.Alt) {
			status = "× BLOCKED / EDGE OR HULL"
			col = Rust
		}
		r.text(xx, y+3, status, col)
	}
	r.text(xx, y+5, action, Muted)
	if hd {
		r.text(103, y+2, "[ SAFETY: DISENGAGED ]", Rust)
		r.text(103, y+3, "AUTHORIZATION: YOUR PROBLEM", Bone)
		r.text(103, y+5, "B  RETURN FROM SHOT REPLAY", Muted)
	}
}
func (r *Raster) board(st State, x, y, cw, ch int, friendly bool, team int) {
	w, h := cw*10, ch*10
	// Thin marks at grid intersections let the sea remain the playing surface.
	for yy := 0; yy <= 10; yy++ {
		for xx := 0; xx <= 10; xx++ {
			r.put(x+xx*cw, y+yy*ch, 0x537477)
		}
	}
	r.line(x, y, x+w, y, 0x708b85)
	r.line(x, y+h, x+w, y+h, 0x365d65)
	for i := 0; i < 10; i++ {
		r.solidText(x+i*cw+cw/2, y/2-1, string(rune('A'+i)), Muted, Ink)
		r.solidText(x-3, y/2+i*ch/2, fmt.Sprintf("%2d", i+1), Muted, Ink)
	}
	if friendly {
		positions := [][2]int{{0, 1}, {2, 4}, {4, 0}, {6, 4}, {8, 1}}
		for i, p := range positions {
			if Scenes[st.Scene].ID == "deployment" && i == 3 {
				continue
			}
			c := assets.Get(fmt.Sprintf("ship-%d-top", team*5+i), "board-"+st.Size())
			r.asset(c, x+p[0]*cw+1, y+p[1]*ch, 0, 0, false)
		}
	} else if Scenes[st.Scene].ID != "deployment" {
		for _, p := range [][2]int{{1, 2}, {3, 7}, {8, 8}, {5, 1}, {0, 5}} {
			px, py := x+p[0]*cw+cw/2, y+p[1]*ch+ch/2
			r.text(px, py/2, "·", Cyan)
		}
		for _, p := range [][2]int{{6, 2}, {6, 3}} {
			px, py := x+p[0]*cw+cw/2, y+p[1]*ch+ch/2
			r.text(px, py/2, "×", Rust)
		}
		tx, ty := x+st.TargetX*cw, y+st.TargetY*ch
		r.line(tx, ty, tx+cw, ty, Gold)
		r.line(tx, ty+ch, tx+cw, ty+ch, Gold)
		if st.Confirmed {
			if st.ShotMiss {
				r.text(tx+cw/2, ty/2, "·", Cyan)
			} else {
				r.text(tx+cw/2, ty/2, "×", Rust)
			}
		} else if !st.SD {
			r.text(tx+cw/2, ty/2, "+", Bone)
		}
	}
	if friendly && Scenes[st.Scene].ID == "deployment" {
		tx, ty := x+st.TargetX*cw, y+st.TargetY*ch
		ww, hh := cw, 4*ch
		if st.Alt {
			ww, hh = cw*4, ch
		}
		legal := placementLegal(st.TargetX, st.TargetY, st.Alt)
		// Four hulls are fixed; the destroyer is the movable placement ghost.
		col := Green
		if !legal {
			col = Rust
		}
		for yy := max(y, ty); yy < min(y+h, ty+hh); yy++ {
			for xx := max(x, tx); xx < min(x+w, tx+ww); xx++ {
				if (xx+yy)%2 == 0 {
					r.blend(xx, yy, col, .65)
				}
			}
		}
	}
}
func (r *Raster) broadcast(st State, s Scene) {
	alien := s.Team == 1
	bg := RGB(0x183035)
	ink := Gold
	if alien {
		bg = 0x2e2536
		ink = Green
	}
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			c := bg
			if y%3 == 0 {
				c = mix(c, Ink, .18)
			}
			r.put(x, y, c)
		}
	}
	cx, cy := r.W/2, int(float64(r.H)*.40)
	rx := r.W / 7
	ry := int(float64(rx) * .7)
	r.ellipse(cx, cy, rx+3, ry+3, mix(ink, Ink, .6), false)
	r.ellipse(cx, cy, rx, ry, ink, false)
	if !alien {
		// An invented naval seal: anchor and angular wings.
		r.line(cx, cy-ry+2, cx, cy+ry-2, ink)
		r.line(cx-4, cy-3, cx+4, cy-3, ink)
		r.line(cx, cy+ry-2, cx-rx+4, cy+2, ink)
		r.line(cx, cy+ry-2, cx+rx-4, cy+2, ink)
		for j := 0; j < 4; j++ {
			r.line(cx-rx-5, cy-ry/2+j*2, cx-3, cy-ry/2+j*2, ink)
			r.line(cx+3, cy-ry/2+j*2, cx+rx+5, cy-ry/2+j*2, ink)
		}
	} else {
		r.ellipse(cx, cy, rx/2, ry, 0x856273, true)
		r.line(cx, cy-ry, cx, cy+ry, ink)
		for j := 0; j < 4; j++ {
			yy := cy - ry + 3 + j*3
			r.line(cx-2, yy, cx-rx, yy-3, ink)
			r.line(cx+2, yy, cx+rx, yy-3, ink)
		}
		r.ellipse(cx, cy, 2, 3, Green, true)
	}
	h := r.H / 2
	row := h - 10
	if st.SD {
		row = h - 9
	}
	r.center(row, short(s.Line, r.W-6), Bone)
	second := "SERVICE GUARANTEES SHORE LEAVE*"
	third := "* Shore availability subject to bombardment."
	if alien {
		second = "WE COME IN PIECES. MOSTLY YOURS."
		third = "This transmission contains 100% recycled threats."
	}
	r.center(row+2, second, ink)
	r.center(row+3, third, Muted)
	if !st.SD {
		r.center(row+5, "PAID FOR BY THE COMMITTEE FOR A LESS INFESTED OCEAN", Muted)
	}
	r.text(3, 3, "● TRANSMITTING", ink)
	r.text(r.W-20, 3, fmt.Sprintf("CH %02d / %02d SEC", s.Team+7, int(st.T)), Muted)
}
func (r *Raster) picker(st State) {
	h := r.H / 2
	w := min(r.W-6, 108)
	hh := min(h-4, 32)
	x, y := (r.W-w)/2, 2
	r.plaque(x, y, w, hh, Ink)
	r.text(x+2, y+1, "SCENE DECK / ENTER TO OPEN / ESC TO CLOSE", Gold)
	visible := hh - 5
	start := max(0, min(st.Selection-visible/2, len(Scenes)-visible))
	for i := start; i < min(len(Scenes), start+visible); i++ {
		s := Scenes[i]
		row := y + 3 + i - start
		col := Muted
		prefix := "  "
		if i == st.Selection {
			r.plaque(x+1, row, w-2, 1, Panel)
			col = Bone
			prefix = "› "
		}
		r.text(x+2, row, short(fmt.Sprintf("%s%02d  %-18s %s", prefix, i+1, s.Category, s.Title), w-4), col)
	}
	r.text(x+2, y+hh-1, "↑ ↓ SELECT    PgUp/PgDn PAGE    HOME/END    ENTER OPEN", Cyan)
}
func (r *Raster) help(st State) {
	w := min(r.W-6, 94)
	h := min(r.H/2-4, 25)
	x := (r.W - w) / 2
	y := 2
	r.plaque(x, y, w, h, Ink)
	lines := []string{"SEA BATTLE 2 / LOOKDEV CONTROLS", "", "Tab              Open the scene deck", "Left / Right     Previous / next scene", "Space            Pause / resume animation", "F                Quarter-speed motion review", "R                Replay current scene", "T                Toggle continuous scene tour", "V                Top / side / front (single-view scenes)", "W                Day / night / squall / spore tide", ". / ,            Step forward / backward 1/10 second", "WASD             Aim on tactical scenes (W moves up there)", "Enter            Play shot outcome from tactical scene", "X                Rotate placement ghost", "B                Return to command station", "F6               Save exact cell snapshot to captures/", "? / Esc          Close help", "Q / Ctrl+C       Exit", "", "Scripted visual studies. No network or full match simulation."}
	for i, line := range lines {
		if i < h-1 {
			r.text(x+2, y+1+i, short(line, w-4), Bone)
		}
	}
}

func placementLegal(x, y int, rotated bool) bool {
	w, h := 1, 4
	if rotated {
		w, h = 4, 1
	}
	if x < 0 || y < 0 || x+w > 10 || y+h > 10 {
		return false
	}
	for _, p := range [][3]int{{0, 1, 2}, {2, 4, 3}, {4, 0, 3}, {8, 1, 5}} {
		if x < p[0]+1 && x+w > p[0] && y < p[1]+p[2] && y+h > p[1] {
			return false
		}
	}
	return true
}
