package visual

import "math"

type sortieState struct {
	age     float64
	win     bool
	high    bool
	bombing bool
}

// Coordinates use the SD half-pixel canvas (80 by 48); HD doubles them.
func (r *Raster) idleLine(x, y, xx, yy float64, c RGB) {
	s := float64(r.W) / 80
	r.line(int(math.Round(x*s)), int(math.Round(y*s)), int(math.Round(xx*s)), int(math.Round(yy*s)), c)
}
func (r *Raster) idleGlow(x, y, radius float64, c RGB, alpha float64) {
	s := float64(r.W) / 80
	cx, cy, rad := int(x*s), int(y*s), max(1, int(radius*s))
	for yy := -rad; yy <= rad; yy++ {
		for xx := -rad; xx <= rad; xx++ {
			d := math.Hypot(float64(xx), float64(yy)) / float64(rad)
			if d < 1 {
				r.blend(cx+xx, cy+yy, c, alpha*(1-d))
			}
		}
	}
}

func (r *Raster) idlePlane(x, y float64, direction int, alien, near bool, t float64) {
	d := float64(direction)
	length := 2.0
	if near {
		length = 3
	}
	if alien {
		// A hooked, beating membrane silhouette with a hot cybernetic eye.
		wing := 1 + .7*math.Sin(t*9)
		r.idleLine(x-length*d, y-wing, x+d, y, 0x779b73)
		r.idleLine(x-length*d, y+wing, x+d, y, 0x557b6e)
		r.idleLine(x-d, y, x+d, y, Green)
	} else {
		r.idleJet(x, y, direction, near, true)
	}
}

// Small bursts keep distant action in perspective instead of borrowing the
// full-screen combat FX. Embers rise and smoke fades at the original impact.
func (r *Raster) idleBurst(x, y, age float64, alien bool) {
	if age < 0 || age > 3 {
		return
	}
	c := Gold
	if alien {
		c = Green
	}
	r.idleGlow(x, y, 2+age, c, math.Max(0, 1-age)*.85)
	for i := 0; i < 7; i++ {
		life := 1.1 + hash(i, 111)
		if age > life {
			continue
		}
		dx := (hash(i, 32) - .5) * 5 * age
		dy := -hash(i, 21)*4*age + age*age
		r.idleLine(x+dx, y+dy, x+dx, y+dy, mix(c, 0x42494b, age/life))
	}
	for i := 0; i < 3; i++ {
		r.idleGlow(x+age*.6+float64(i)*.4, y-age*1.3, 1+age*.6, 0x41404a, .3*(1-age/3))
	}
}

func (r *Raster) idleShore(age float64, n int) {
	if age >= 0 && age < 5 {
		// Sample the same land profile as ocean(), and choose actual land.
		x := int(3 + hash(n, 57)*74)
		for i := 0; i < 80; i++ {
			a := math.Sin(float64(x)*.08) + .5*math.Sin(float64(x)*.21)
			if a > .5 {
				// Match the native shoreline, including HD's finer rounding.
				px := int(float64(x) * float64(r.W) / 80)
				land := math.Sin(float64(px)*.08) + .5*math.Sin(float64(px)*.21)
				if land > .5 {
					y := float64(int(float64(r.H)*.43)-int(land*float64(r.W)/65)) * 80 / float64(r.W)
					r.idleBurst(float64(x), y, age, false)
					r.idleBurst(float64(x)+2, y, age-.8, false)
					break
				}
			}
			x = (x + 1) % 80
		}
	}
}

func (r *Raster) idleLeviathan(age float64) {
	if age >= 0 && age < 11 {
		rise := smooth(age/3) * (1 - smooth((age-6)/5))
		for i := 0; i <= 26; i++ {
			x := 25 + float64(i) + age*.3
			y := 26 - math.Sin(float64(i)/26*math.Pi)*6*rise
			r.idleLine(x, y, x, 26, mix(0x1a333b, 0x39564f, rise))
			if i%3 == 0 {
				r.idleLine(x, y, x+.5, y-rise*1.5, mix(0x29474a, Green, rise*.6))
			}
		}
		r.idleGlow(48+age*.3, 24, 2, Green, rise*.65)
		r.idleLine(23+age*.3, 27, 55+age*.3, 27, mix(0x243f49, Cyan, rise*.35))
	}
}

func (r *Raster) idleCarrierCue(age float64, radar bool) {
	if age < 0 || age > 5 {
		return
	}
	if radar {
		// Radar dish rotates on the island, with a brief running-light wink.
		x := 51 + math.Sin(age*2)*2
		r.idleLine(51, 20, x, 18, Muted)
		if math.Mod(age, 1.2) < .2 {
			r.idleGlow(53, 24, 2, Rust, .75)
		}
	} else {
		// Defensive burst from the bow: three rounds and a far interception.
		for i := 0; i < 3; i++ {
			a := age - float64(i)*.35
			if a >= 0 && a < 1.1 {
				x, y := 17-a*12, 28-a*7
				r.idleLine(x, y, x+1.5, y+.8, Gold)
				if a < .15 {
					r.idleGlow(17, 28, 2, Gold, .8)
				}
			}
		}
		r.idleBurst(4, 20, age-1.1, true)
	}
}

func (r *Raster) idleSortie(s sortieState) {
	if s.age < 0 {
		return
	}
	if s.age < sortieLiftTime {
		r.idleElevator(s.age, false)
		return
	}
	a := s.age - sortieLiftTime
	t := a
	switch {
	case s.win && a >= 47:
		r.idleElevator(a-47, true)
	case a >= 0 && a < 5:
		// Catapult roll, then climb out of the frame to port.
		if a < .35 {
			// The stationary tension beat has no velocity to infer heading from.
			// Keep the same bow-facing orientation as the elevator and launch.
			r.idleJet(30, 27, -1, true, true)
		} else {
			r.flyingJet(flightAt(launchPath, a, 0), true, true)
		}
		if a >= .35 && a < 2.3 {
			// Steam lingers along the catapult after the aircraft has cleared it.
			fade := 1 - smooth((a-.8)/1.5)
			for x := 9.0; x <= 29; x += 4 {
				if x >= launchPath(a).x {
					r.idleGlow(x, 28-(a-.35)*.7, 1.6, Bone, .45*fade)
				}
			}
		}
	case a >= 10 && a < 17:
		path := func(t float64) flightPoint {
			p := (t - 10) / 7
			return flightPoint{lerp(-6, 86, p), 18 + math.Sin(p*math.Pi)}
		}
		r.flyingJet(flightAt(path, a, 0), false, true)
	case s.bombing && a >= 22 && a < 38:
		r.idleBombingRun(a)
	case a >= 22 && a < 29:
		path := func(t float64) flightPoint { return dogfightPath(t, s.high) }
		r.flyingJet(flightAt(path, a, .15*math.Sin((a-22)/7*math.Pi)), false, true)
		enemy := pursuitPath(a, s.high)
		r.idlePlane(enemy.x, enemy.y, -1, true, false, t)
		if math.Mod(a, .8) < .18 {
			p := path(a)
			r.idleLine(enemy.x-3, enemy.y, p.x+3, p.y, Green)
		}
	case a >= 29 && a < 38:
		if s.win {
			path := func(t float64) flightPoint { return dogfightPath(t, s.high) }
			bank := math.Sin(math.Pi * clamp((a-29)/3, 0, 1))
			r.flyingJet(flightAt(path, a, bank), false, true)
			const hitTime = 32.45
			impact := pursuitPath(hitTime, s.high)
			if a < hitTime {
				enemy := pursuitPath(a, s.high)
				dir := -1
				if pursuitPath(a+.01, s.high).x > enemy.x {
					dir = 1
				}
				r.idlePlane(enemy.x, enemy.y, dir, true, false, t)
			}
			if a >= 32 && a < hitTime && math.Mod(a, .15) < .08 {
				p := path(a)
				r.idleLine(p.x+3, p.y, impact.x, impact.y, Gold)
			}
			r.idleBurst(impact.x, impact.y, a-hitTime, true)
		} else {
			r.idleBurst(15, 18, a-29, false)
			enemy := pursuitPath(29, s.high)
			r.idlePlane(enemy.x-(a-29)*11, enemy.y, -1, true, false, t)
		}
	case s.win && a >= 39 && a < 47:
		r.landingJet(a)
	}
}

func (r *Raster) menuTitle(row int, age float64) {
	// A separate mask ensures the glint touches lettering only, never the sky.
	mask := New(r.W, r.H/2)
	mask.bigWord("SEA BATTLE II", row, Bone)
	scale := float64(r.W) / 80
	beam := (age/2.2)*float64(r.W+16) - 8
	for y := row * 2; y < min(r.H, row*2+int(5*scale)); y++ {
		for x := 0; x < r.W; x++ {
			if mask.Pixels[y*r.W+x] == 0 {
				continue
			}
			base := mix(Bone, 0x87979d, float64(y-row*2)/(5*scale)*.32)
			if age >= 0 && age < 2.2 {
				d := math.Abs(float64(x) + float64(y-row*2)*.6 - beam)
				base = mix(base, 0xffffff, math.Max(0, 1-d/(3*scale))*.9)
			}
			r.put(x, y, base)
		}
	}
}
