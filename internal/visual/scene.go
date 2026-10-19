package visual

import (
	"fmt"
	"math"
	"sea-battle-2/internal/assets"
	"strings"
)

func Render(st State) *Frame {
	w, h := 160, 48
	if st.SD {
		w, h = 80, 24
	}
	r := New(w, h)
	s := Scenes[st.Scene]
	weather := s.Weather
	if st.Weather >= 0 {
		weather = st.Weather
	}
	camera := s.Camera
	if st.Camera != "" {
		camera = st.Camera
	}
	if s.Kind == Spectacle {
		r.spectacle(st, s, weather)
	} else if s.Kind == Fleet {
		r.fleet(st, s, weather)
	} else if s.Kind == Duel {
		r.duel(st, s, weather)
	} else if s.Kind == AirOps {
		r.airOps(st, s, weather)
	} else if s.Kind == Scope {
		r.scope(st, s, weather)
	} else if s.Kind == Tactical {
		r.tactical(st, s, weather)
	} else if s.Kind == Broadcast {
		r.broadcast(st, s)
	} else {
		r.cinema(st, s, weather, camera)
	}
	r.chrome(st, s)
	if st.Picker {
		r.picker(st)
	}
	if st.Help {
		r.help(st)
	}
	return r.frame()
}
func (r *Raster) chrome(st State, s Scene) {
	h := r.H / 2
	accent := Gold
	if s.Team == 1 {
		accent = Green
	}
	r.plaque(0, 0, r.W, 2, Ink)
	r.text(2, 0, "SEA BATTLE 2", Bone)
	info := "LOOKDEV / " + st.Size()
	if st.Tour {
		info += " / TOUR"
	}
	if st.Paused {
		info += " / PAUSED"
	}
	if st.Slow {
		info += " / 1/4"
	}
	r.text(r.W-len(info)-2, 0, info, Muted)
	title := s.Category + "  /  " + s.Title
	if len(title) > r.W-4 {
		title = s.Title
	}
	r.text(2, 1, title, accent)
	r.plaque(0, h-2, r.W, 2, Ink)
	footer := "[Tab] SCENES   [← →] CUT   [Space] PAUSE   [R] REPLAY   [T] TOUR   [V] CAMERA   [W] SEA   [?] HELP   [Q] EXIT"
	if st.SD {
		footer = "Tab SCENES  ← → CUT  Space PAUSE  R REPLAY  T TOUR  ? HELP  Q EXIT"
	}
	r.text(2, h-1, footer, Muted)
	n := fmt.Sprintf("%02d/%02d", st.Scene+1, len(Scenes))
	r.text(2, h-2, n, accent)
	line := s.Line
	if st.SD && len(line) > r.W-14 {
		line = short(line, r.W-14)
	}
	r.text(10, h-2, line, Bone)
}
func short(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:max(0, n-3)] + "..."
}
func (r *Raster) cinema(st State, s Scene, weather int, camera string) {
	r.cinemaScene(st, s, weather, camera, true)
}
func (r *Raster) cinemaScene(st State, s Scene, weather int, camera string, showHUD bool) {
	t := st.T
	scale := float64(r.W) / 160
	top := camera == "TOP"
	r.ocean(t, weather, top)
	c := assets.Get(fmt.Sprintf("ship-%d-%s", s.Ship, strings.ToLower(camera)), st.Size())
	x := (r.W - c.W) / 2
	y := int(float64(r.H)*.64) - int(float64(c.H)*.87)
	if top {
		y = min((r.H-c.H)/2, (r.H/2-7)*2-c.H-1)
		if !st.SD {
			y = min((r.H-c.H)/2, (r.H/2-9)*2-c.H-1)
		}
		x = r.W/2 - c.W/2
	}
	bob := int(math.Round(math.Sin(t*1.2) * scale))
	y += bob
	// Keep wide hero hulls out of the title/console safe areas in both sizes.
	waterline := y + int(float64(c.H)*.87)
	tint := RGB(0)
	amount := 0.0
	if weather == 1 {
		tint = 0x28435c
		amount = .18
	}
	if weather == 3 {
		tint = 0x5a8269
		amount = .08
	}
	if s.Kind == Arrival {
		x += int((1 - smooth(t/5)) * float64(r.W) * .2)
	}
	if s.Kind == Depth {
		r.underwater(st, s)
		return
	}
	sink := 0.0
	if s.Kind == Sink {
		sink = smooth((t - 4) / 6)
	}
	if !top {
		r.reflection(c, x, waterline+2, t)
		r.wake(x+c.W/2, waterline+1, c.W-5, t, environments[weather].crest)
	}
	// Sinking clips against a fixed waterline, preserving ocean in front of the hull.
	if s.Kind == Sink {
		drop := int(sink * float64(c.H))
		for yy := 0; yy < c.H; yy++ {
			for xx := 0; xx < c.W; xx++ {
				py := y + yy + drop + int(float64(xx-c.W/2)*sink*.16)
				n := c.Pixels[yy*c.W+xx]
				if n > 0 && py <= waterline {
					r.put(x+xx, py, mix(RGB(assets.Palette[n]), environments[weather].deep, sink*.5))
				}
			}
		}
	} else {
		r.asset(c, x, y, tint, amount, false)
	}
	impactX := x + int(float64(c.W)*.61)
	impactY := waterline - int(7*scale)
	phase := ""
	status := Cyan
	switch s.Kind {
	case Arrival:
		logoRow := 3
		subtitleRow := 7
		if !st.SD {
			logoRow = 5
			subtitleRow = 12
		}
		r.bigWord("SEA BATTLE II", logoRow, Bone)
		r.center(subtitleRow, "OPERATION: UNEXPECTED GUESTS", Gold)
		phase = "UNIDENTIFIED CONTACTS / RANGE CLOSING"
		if t > 5 {
			phase = "OPERATION: UNEXPECTED GUESTS"
		}
		// Small hostile silhouettes on the far horizon.
		for i := 0; i < 4; i++ {
			xx := r.W/8 + i*r.W/8
			r.line(xx, int(float64(r.H)*.45), xx+int(5*scale), int(float64(r.H)*.45), 0x233a3d)
		}
	case Portrait:
		phase = strings.ToUpper(assets.Names[s.Ship]) + " / " + camera
		if !st.SD {
			r.text(4, 4, fmt.Sprintf("HULL %02d", s.Ship+1), Muted)
			r.text(4, 5, fmt.Sprintf("%d OCCUPIED SQUARES", assets.Lengths[s.Ship]), Bone)
		}
	case Launch:
		phase = "FIRE CONTROL / SOLUTION LOCKED"
		if t >= 2 && t < 5.5 {
			phase = "WEAPON AWAY"
			age := t - 2
			px := float64(x+c.W*3/4) + age*float64(r.W)*.22
			py := float64(waterline) - age*float64(r.H)*.32
			col := Gold
			if s.Team == 1 {
				col = Green
			}
			for j := 0; j < 10; j++ {
				q := float64(j) * scale
				r.blend(int(px-q), int(py+q*.7), col, 1-float64(j)/10)
			}
			r.ellipse(int(px), int(py), max(1, int(2*scale)), max(1, int(scale)), Bone, true)
			if age < .55 {
				r.explosion(x+c.W*3/4, waterline-int(7*scale), age, s.Team == 1, scale*.6)
			}
		}
		if t >= 5.5 {
			phase = "INBOUND / STAND BY FOR IMPACT"
		}
	case Miss:
		phase = fmt.Sprintf("TERMINAL APPROACH / GRID %c%d", 'A'+st.TargetX, st.TargetY+1)
		if t > 1.2 {
			phase = "MISS / NO CONTACT"
			r.splash(r.W/3, waterline+int(2*scale), t-1.2, scale)
		}
	case Hit, Sink:
		phase = fmt.Sprintf("TERMINAL APPROACH / GRID %c%d", 'A'+st.TargetX, st.TargetY+1)
		if t < 1.5 {
			age := 1.5 - t
			px := float64(impactX) - age*float64(r.W)*.35
			py := float64(impactY) - age*float64(r.H)*.4
			col := Gold
			if s.Team == 1 {
				col = Green
			}
			r.line(int(px), int(py), int(px-4*scale), int(py-4*scale), col)
		}
		if t >= 1.5 {
			phase = "HIT CONFIRMED / HULL BREACH"
			status = Rust
			r.explosion(impactX, impactY, t-1.5, s.Team == 1, scale)
			r.burst(impactX, impactY, t-1.5, st.Size(), s.Team == 1)
			if sink < .72 {
				r.smoke(impactX, impactY+int(sink*float64(c.H)), t, s.Team == 1, scale)
				r.fire(impactX, impactY+int(sink*float64(c.H)), t, s.Team == 1, scale)
			}
		}
		if s.Kind == Sink && t > 4 {
			phase = "CRITICAL FLOODING / VESSEL LOST"
			r.splash(x+c.W/3, waterline, t-4, scale)
			if t > 8 {
				phase = "SUNK / CONTACT REMOVED"
				r.ellipse(impactX, waterline, max(1, int(8*scale)), 1, 0x506b67, false)
			}
		}
	case Result:
		phase = "ALL ENEMY VESSELS ACCOUNTED FOR"
		if s.Team == 1 {
			phase = "ALL HUMAN VESSELS CONSUMED"
		}
		for i := 0; i < 18; i++ {
			xx := int(hash(i, 91) * float64(r.W))
			yy := int(math.Mod(hash(i, 92)*float64(r.H)+t*3, float64(r.H)))
			if yy > 5 && yy < r.H-12 {
				r.put(xx, yy, Gold)
			}
		}
	}
	r.rain(t, weather)
	if !showHUD {
		return
	}
	if s.Kind == Result {
		word := "VICTORY"
		if s.Team == 1 {
			word = "DEFEAT"
		}
		r.bigWord(word, 3, Gold)
	} else if !st.SD {
		r.text(4, 3, "PACIFIC THEATER  /  SECTOR 09", Muted)
		r.text(r.W-27, 3, []string{"DAY / SEA STATE 02", "NIGHT / SEA STATE 01", "SQUALL / SEA STATE 05", "SPORE TIDE / UNKNOWN"}[weather], Muted)
	}
	r.cinemaInstruments(st, s, phase, status)
}
func (r *Raster) cinemaInstruments(st State, s Scene, phase string, status RGB) {
	h := r.H / 2
	y := h - 7
	if !st.SD {
		y = h - 9
	}
	if s.Team == 0 {
		r.console(2, y, r.W-4, h-y-2, false)
		r.text(4, y, " FIRE CONTROL / OPTICAL RELAY ", Gold)
		r.text(4, y+2, short(phase, r.W-32), status)
		if !st.SD {
			r.text(4, y+4, "BEARING  027°     RANGE  4.8 NM     WIND  12 KTS", Muted)
			r.text(4, y+5, "MUNITIONS AUTHORIZED BY SOMEONE ELSE", Bone)
		}
		r.gauge(r.W-15, (y+3)*2, st.T, Gold, false)
		if !st.SD {
			r.instrumentBank(82, y+2, st.T, false)
		}
	} else {
		r.console(2, y, r.W-4, h-y-2, true)
		for x := 3; x < r.W-2; x += 7 {
			r.line(x, y*2, x+2, y*2-2, 0x886775)
			r.line(x+2, y*2-2, x+4, y*2, 0x5e4355)
		}
		r.text(5, y, " SYNAPTIC LINK / BROOD CONSENSUS ", Green)
		r.text(5, y+2, short(phase, r.W-32), status)
		if !st.SD {
			r.text(5, y+4, "HUNGER  ELEVATED     ENZYME  ARMED     MERCY  ABSENT", Muted)
			r.text(5, y+5, "THE QUEEN HAS ALREADY APPROVED THIS MESSAGE", Bone)
		}
		r.gauge(r.W-15, (y+3)*2, st.T, Green, true)
		if !st.SD {
			r.instrumentBank(82, y+2, st.T, true)
		}
	}
}
func (r *Raster) gauge(x, y int, t float64, c RGB, alien bool) {
	radius := 4
	if r.W > 80 {
		radius = 6
	}
	r.ellipse(x, y, radius, radius, 0x08151a, true)
	r.ellipse(x, y, radius, radius, mix(c, Ink, .55), false)
	if alien {
		for j := 0; j < 7; j++ {
			yy := y - radius + j*2
			r.line(x-3, yy, x+int(math.Sin(t*2+float64(j))*3), yy, c)
		}
	} else {
		for j := 0; j < 7; j++ {
			a := math.Pi * (.85 + float64(j)*.21)
			r.put(x+int(math.Cos(a)*float64(radius-1)), y+int(math.Sin(a)*float64(radius-1)), c)
		}
		a := math.Pi * (1.2 + .15*math.Sin(t*.8))
		r.line(x, y, x+int(math.Cos(a)*float64(radius-2)), y+int(math.Sin(a)*float64(radius-2)), Bone)
		r.put(x, y, c)
	}
}
func (r *Raster) underwater(st State, s Scene) {
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			v := float64(y) / float64(r.H)
			c := mix(0x19444a, 0x040e1b, v)
			ray := math.Sin(float64(x)*.065 + float64(y)*.05 + st.T*.2)
			if ray > .8 {
				c = mix(c, 0x6b9c81, (ray-.8)*.2)
			}
			r.put(x, y, c)
		}
	}
	c := assets.Get("ship-6-side", st.Size())
	x := (r.W-c.W)/2 + int(math.Sin(st.T*.3)*3)
	y := r.H/2 - c.H/2
	r.asset(c, x, y, 0x102a38, .38, false)
	for i := 0; i < 25; i++ {
		xx := int(hash(i, 20) * float64(r.W))
		yy := int(math.Mod(hash(i, 9)*float64(r.H)-st.T*2+float64(r.H)*10, float64(r.H)))
		r.put(xx, yy, 0x386465)
	}
	// Slow sonar pulse crosses the creature; it never flashes the whole screen.
	cx := r.W / 2
	radius := int(math.Mod(st.T*12, float64(r.W)))
	for a := 0; a < 360; a += 2 {
		v := float64(a) * math.Pi / 180
		r.blend(cx+int(math.Cos(v)*float64(radius)), r.H/2+int(math.Sin(v)*float64(radius)), Green, .35)
	}
	r.cinemaInstruments(st, s, "PASSIVE CONTACT / DO NOT PING AGAIN", Green)
}

var letters = map[rune][]string{
	'W': {"#   #", "#   #", "# # #", "## ##", "#   #"},
	'H': {"#   #", "#   #", "#####", "#   #", "#   #"},
	'M': {"#   #", "## ##", "# # #", "#   #", "#   #"},
	'U': {"#   #", "#   #", "#   #", "#   #", " ### "},
	'N': {"#   #", "##  #", "# # #", "#  ##", "#   #"},
	'K': {"#   #", "#  # ", "###  ", "#  # ", "#   #"},
	'!': {"#", "#", "#", " ", "#"},
	' ': {"   ", "   ", "   ", "   ", "   "},
	'2': {"#### ", "    #", " ### ", "#    ", "#####"},
	'S': {" ####", "#    ", " ### ", "    #", "#### "},
	'B': {"#### ", "#   #", "#### ", "#   #", "#### "},
	'L': {"#    ", "#    ", "#    ", "#    ", "#####"},
	'V': {"#   #", "#   #", "#   #", " # # ", "  #  "}, 'I': {"###", " # ", " # ", " # ", "###"}, 'C': {" ###", "#   ", "#   ", "#   ", " ###"}, 'T': {"#####", "  #  ", "  #  ", "  #  ", "  #  "}, 'O': {" ### ", "#   #", "#   #", "#   #", " ### "}, 'R': {"#### ", "#   #", "#### ", "#  # ", "#   #"}, 'Y': {"#   #", " # # ", "  #  ", "  #  ", "  #  "}, 'D': {"#### ", "#   #", "#   #", "#   #", "#### "}, 'E': {"#####", "#    ", "#### ", "#    ", "#####"}, 'F': {"#####", "#    ", "#### ", "#    ", "#    "}, 'A': {" ### ", "#   #", "#####", "#   #", "#   #"}}

func (r *Raster) bigWord(s string, row int, c RGB) {
	size := 1
	if r.W > 80 {
		size = 2
	}
	width := 0
	for _, ch := range s {
		width += (len(letters[ch][0]) + 1) * size
	}
	x := (r.W - width) / 2
	for _, ch := range s {
		for y, line := range letters[ch] {
			for xx, v := range line {
				if v == '#' {
					r.rect(x+xx*size, row*2+y*size, size, size, c)
				}
			}
		}
		x += (len(letters[ch][0]) + 1) * size
	}
}
