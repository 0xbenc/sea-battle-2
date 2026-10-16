package visual

import (
	"fmt"
	"git.occ.farm/occ/dotmason/art"
	"math"
	"sea-battle-2/internal/assets"
	"strings"
)

func (r *Raster) burst(x, y int, age float64, size string, alien bool) {
	if age < 0 || age > 1.6 {
		return
	}
	c := assets.Get("impact", size)
	track, _ := c.Track("burst")
	f, ok := track.AtUnit(uint64(age * 10))
	if !ok {
		return
	}
	f.Samples(func(p art.Sample) {
		color := RGB(assets.Palette[p.FG])
		if alien && p.FG == 20 {
			color = Green
		}
		r.blend(x+p.X-c.Anchor.X, y+p.Y-c.Anchor.Y, color, 1-age/1.7)
	})
}
func (r *Raster) fleet(st State, s Scene, weather int) {
	r.ocean(st.T, weather, true)
	row := 3
	if !st.SD {
		row = 8
	}
	ids := []string{"CYCLONE", "VIRGINIA", "FREEDOM", "ZUMWALT", "FORD"}
	if s.Team == 1 {
		ids = []string{"DRONE", "LURKER", "RAVAGER", "CARAPACE", "HIVE"}
	}
	for i := 0; i < 5; i++ {
		ship := s.Team*5 + i
		c := assets.Get(fmt.Sprintf("ship-%d-top", ship), "fleet-"+st.Size())
		cx := (i*2 + 1) * r.W / 10
		y := row*2 + (5-assets.Lengths[ship])*2
		// Wakes narrow behind each hull; no runtime sprite scaling.
		for j := 0; j < 7; j++ {
			yy := y + c.H + j
			xoff := int(float64(j) * .45)
			r.blend(cx-c.W/3-xoff, yy, 0x86afa6, .45)
			r.blend(cx+c.W/3+xoff, yy, 0x86afa6, .45)
		}
		r.asset(c, cx-c.W/2, y+int(math.Sin(st.T+float64(i))*.7), 0, 0, false)
		labelrow := r.H/2 - 7
		r.solidText(cx-len(ids[i])/2, labelrow, ids[i], Bone, Ink)
		r.solidText(cx-2, labelrow+1, strings.Repeat("▪", assets.Lengths[ship]), Gold, Ink)
	}
	r.center(2, "FLEET RECOGNITION / ALL FIVE HULLS", Muted)
	r.plaque(3, r.H/2-4, r.W-6, 2, Ink)
	caption := "STEEL / RADAR / GUIDED BAD DECISIONS"
	if s.Team == 1 {
		caption = "CHITIN / GRAFTED CIRCUITRY / NO TABLE MANNERS"
	}
	r.center(r.H/2-4, caption, Green)
}
func (r *Raster) duel(st State, s Scene, weather int) {
	t := st.T
	if t < 4 {
		ss := s
		ss.Kind = Launch
		local := st
		local.T = t
		ss.Title = "WEAPON AUTHORIZED"
		r.cinema(local, ss, weather, "SIDE")
		r.cutLabel(st, "01 / LAUNCH PLATFORM")
	} else if t < 7 {
		r.ocean(t, weather, false)
		sc := float64(r.W) / 160
		cx, cy := r.W/2, int(float64(r.H)*.4)
		alien := s.Team == 1
		// Track the projectile: sea streaks move, projectile holds the composition.
		for j := 0; j < 25; j++ {
			xx := int(math.Mod(hash(j, 3)*float64(r.W)-t*float64(r.W)*.6+float64(r.W)*40, float64(r.W)))
			yy := int(float64(r.H)*.5 + hash(j, 41)*float64(r.H)*.3)
			r.line(xx, yy, xx+int(8*sc), yy, 0x41636a)
		}
		if !alien {
			r.rect(cx-int(13*sc), cy-int(2*sc), int(25*sc), max(2, int(4*sc)), 0xb1b8a6)
			r.line(cx+int(12*sc), cy-int(sc), cx+int(17*sc), cy, Gold)
			r.line(cx-int(5*sc), cy, cx-int(12*sc), cy-int(7*sc), 0x7c9294)
			r.line(cx-int(5*sc), cy, cx-int(12*sc), cy+int(7*sc), 0x7c9294)
			for i := 0; i < 20; i++ {
				r.blend(cx-int(13*sc)-i, cy+int(math.Sin(float64(i)+t*12)*sc), mix(Bone, Rust, float64(i)/20), 1-float64(i)/20)
			}
		} else {
			r.ellipse(cx, cy, int(12*sc), int(5*sc), 0x8b6f79, true)
			r.ellipse(cx+int(6*sc), cy, int(3*sc), int(2*sc), Green, true)
			for j := 0; j < 4; j++ {
				for i := 0; i < 18; i++ {
					xx := cx - int(10*sc) - i
					yy := cy + int(math.Sin(float64(i)*.3-t*5+float64(j))*float64(j+1)*sc)
					r.put(xx, yy, mix(0x587867, Green, .4))
				}
			}
		}
		r.cinemaInstruments(st, s, "INBOUND / PAYLOAD VERY COMMITTED", Gold)
		r.cutLabel(st, "02 / TERMINAL PURSUIT")
	} else if t < 12 {
		ss := s
		ss.Kind = Hit
		if st.ShotMiss {
			ss.Kind = Miss
		}
		ss.Ship = 8
		if s.Team == 1 {
			ss.Ship = 3
		}
		local := st
		local.T = t - 7 + 1
		r.cinema(local, ss, weather, "FRONT")
		r.cutLabel(st, "03 / IMPACT CAMERA")
	} else {
		ss := s
		ss.Kind = Tactical
		st.Confirmed = true
		r.tactical(st, ss, weather)
		label := "04 / HIT CONFIRMED. NEXT SHOT."
		if st.ShotMiss {
			label = "04 / MISS CONFIRMED. NEXT SHOT."
		}
		if st.SD {
			r.solidText(3, 2, label, Gold, Ink)
		} else {
			r.cutLabel(st, label)
		}
	}
}
func (r *Raster) cutLabel(st State, s string) {
	row := 3
	if !st.SD {
		row = 6
	}
	r.solidText(3, row, s, Gold, Ink)
}
func (r *Raster) airOps(st State, s Scene, weather int) {
	r.ocean(st.T, weather, true)
	c := assets.Get(fmt.Sprintf("ship-%d-top", s.Ship), st.Size())
	x := r.W/2 - c.W/2
	y := max(5, (r.H-c.H)/2-3)
	if st.SD {
		y = min(y, (r.H/2-7)*2-c.H-1)
	}
	r.asset(c, x, y, 0, 0, false)
	sc := float64(r.W) / 160
	for i := 0; i < 5; i++ {
		age := st.T - 1 - float64(i)*1.25
		if age < 0 {
			continue
		}
		px := float64(r.W)/2 + float64(i-2)*sc*2 + math.Sin(age*.7)*age*3*sc
		py := float64(y+c.H/2) - age*age*4*sc
		if s.Team == 0 {
			xx, yy := int(px), int(py)
			r.line(xx, yy+int(4*sc), xx, yy-int(4*sc), Bone)
			r.line(xx-int(4*sc), yy+int(sc), xx+int(4*sc), yy+int(sc), Bone)
			r.line(xx-int(2*sc), yy+int(3*sc), xx+int(2*sc), yy+int(3*sc), Muted)
			r.line(xx, yy+int(5*sc), xx, yy+int(9*sc), Gold)
		} else {
			xx, yy := int(px), int(py)
			wing := int((3 + math.Sin(age*12)*2) * sc)
			r.ellipse(xx, yy, max(1, int(2*sc)), max(1, int(3*sc)), 0xa87b81, true)
			r.line(xx-wing, yy+int(sc), xx, yy-int(sc), Green)
			r.line(xx+wing, yy+int(sc), xx, yy-int(sc), Green)
		}
	}
	r.cinemaInstruments(st, s, "FLIGHT OPERATIONS / DECK CLEAR", Cyan)
	r.cutLabel(st, "LAUNCH WINDOW / KEEP YOUR HEAD DOWN")
}
func (r *Raster) scope(st State, s Scene, weather int) {
	r.ocean(st.T, weather, false)
	c := assets.Get("ship-9-side", "SD")
	x := (r.W-c.W)/2 + int(math.Sin(st.T*.4)*4)
	y := int(float64(r.H)*.51) - c.H/2
	r.asset(c, x, y, 0x3c6d63, .5, false)
	cx, cy := r.W/2, int(float64(r.H)*.46)
	rx, ry := r.W/2-6, int(float64(r.H)*.32)
	for yy := 4; yy < r.H-4; yy++ {
		for xx := 0; xx < r.W; xx++ {
			d := float64((xx-cx)*(xx-cx))/float64(rx*rx) + float64((yy-cy)*(yy-cy))/float64(ry*ry)
			if d > 1 {
				r.put(xx, yy, Ink)
			} else {
				r.blend(xx, yy, 0x649c79, .10)
			}
		}
	}
	r.line(cx-rx+8, cy, cx-5, cy, 0x78a08a)
	r.line(cx+5, cy, cx+rx-8, cy, 0x78a08a)
	r.line(cx, cy-ry+5, cx, cy-4, 0x78a08a)
	r.line(cx, cy+4, cx, cy+ry-4, 0x78a08a)
	for i := -3; i <= 3; i++ {
		if i != 0 {
			r.line(cx+i*6, cy-1, cx+i*6, cy+1, Green)
		}
	}
	r.cinemaInstruments(st, s, "CONTACT 027 / HIVE CARRIER / HOLD", Green)
	r.cutLabel(st, "OPTICAL / 4× / PASSIVE")
}

func (r *Raster) spectacle(st State, s Scene, weather int) {
	r.ocean(st.T, weather, false)
	sc := float64(r.W) / 160
	cx := r.W / 2
	waterline := int(float64(r.H) * .65)
	c := assets.Get(fmt.Sprintf("ship-%d-side", s.Ship), "SD")
	age := st.T - 3
	if age < .5 {
		r.asset(c, cx-c.W/2, waterline-int(float64(c.H)*.87), 0x305057, .2, false)
	}
	phase := "ONE CONTACT / ALL AVAILABLE FIREPOWER"
	if age < 0 {
		// A deliberately oversized delivery, framed against a small distant victim.
		py := waterline - int(-age*float64(r.H)*.35)
		col := Gold
		if s.Team == 1 {
			col = Green
		}
		r.ellipse(cx, py, max(2, int(4*sc)), max(3, int(9*sc)), col, true)
		for j := 0; j < 10; j++ {
			r.line(cx-1, py-j*2, cx+1, py-j*2, mix(col, Ink, float64(j)/10))
		}
	} else {
		phase = "HIT / THE SQUARE IS STILL THE SAME SIZE"
		col, edge := RGB(0xffe3a5), RGB(0xc46c4c)
		if s.Team == 1 {
			col = 0xc6ffc1
			edge = 0x55a99c
		}
		rise := smooth(age / 2.3)
		fade := smooth((age - 5) / 5)
		// Tall expanding water/energy column; no full-screen flash.
		for yy := 0; yy < int(35*sc*rise); yy++ {
			v := float64(yy) / (35 * sc)
			width := int((3 + 8*math.Sin(v*math.Pi) + 5*hash(yy, 30)) * sc * (1 - fade*.5))
			for xx := -width; xx <= width; xx++ {
				t := math.Abs(float64(xx)) / float64(max(1, width))
				cc := mix(col, edge, t)
				r.blend(cx+xx, waterline-yy, cc, (1-fade)*.95)
			}
		}
		capY := waterline - int(31*sc*rise)
		rx := int((4 + age*6) * sc)
		rx = min(rx, int(40*sc))
		ry := max(2, int((3+rise*5)*sc))
		if age < 10 {
			for j := 0; j < 7; j++ {
				xx := cx + int(float64(j-3)*float64(rx)/4)
				yy := capY + int(math.Sin(float64(j)*1.7)*3*sc)
				r.fadingEllipse(xx, yy, max(1, rx/3), ry, mix(col, 0x465a5a, fade*.8+float64(j%3)*.12), 1-fade)
			}
		}
		for ring := 0; ring < 3; ring++ {
			a := age - float64(ring)*.4
			if a >= 0 {
				r.splash(cx, waterline, a, sc*1.3)
			}
		}
		for i := 0; i < 45; i++ {
			v := hash(i, 51)
			px := cx + int((v-.5)*float64(r.W)*age*.3)
			py := waterline - int((age*(12+hash(i, 52)*18)-age*age*5)*sc)
			if age < 4.5 {
				r.put(px, py, mix(col, edge, v))
			}
		}
		if age > 8 {
			phase = "CONTACT LOST / ACCOUNTING NOTIFIED"
		}
	}
	r.cinemaInstruments(st, s, phase, Gold)
	r.cutLabel(st, "ORDNANCE RELEASE / PROPORTIONALITY WAIVER APPROVED")
}
