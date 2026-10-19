package visual

import "math"

// Brood ballistic submarine: a low, segmented Deep Lurker relative with
// dorsal launch sacs. Its purple chitin and mint grafts match the fleet art.
func (r *Raster) idleMissileSub(age float64) {
	if age < 0 || age > 19 {
		return
	}
	rise := smooth(age/2.8) * (1 - smooth((age-10)/4))
	const cx, water = 68.0, 24.0
	if rise > 0 {
		hull := New(r.W, r.H/2)
		scale := r.W / 80
		for column := -9 * scale; column <= 9*scale; column++ {
			i := float64(column) / float64(scale)
			arch := math.Sqrt(math.Max(0, 1-i*i/81))
			y := water - 3.5*arch*rise
			hull.idleLine(cx+i, y, cx+i, water, mix(0x342d43, 0x946d83, arch*.7))
			if column%(3*scale) == 0 && i > -8 && i < 8 {
				hull.idleLine(cx+i, y, cx+i-1, y-1.6*rise, 0xb5ae8b)
				hull.idleGlow(cx+i, y+1, 1.5, 0x70e2ba, .6*rise)
			}
		}
		hull.idleLine(cx+7, water-1, cx+8, water-1, Green)
		// Bow mandibles and water draining off the dorsal shell.
		hull.idleLine(cx-8, water-1, cx-11, water-2*rise, 0x92728a)
		hull.idleLine(cx-8, water, cx-11, water+rise, 0x92728a)
		hull.idleLine(cx-11, water+1, cx+11, water+1, mix(0x243f49, Cyan, rise*.4))
		for i, color := range hull.Pixels {
			if color != 0 {
				r.blend(i%r.W, i/r.W, color, rise)
			}
		}
	}
	for i := 0; i < 3; i++ {
		flight := age - 3.5 - float64(i)*1.15
		startX, targetX := cx+float64(i-1)*3, 10+float64(i)*2
		// Use the renderer's native land height, so SD and HD impacts hit land.
		px := targetX * float64(r.W) / 80
		land := math.Max(0, math.Sin(px*.08)+.5*math.Sin(px*.21))
		targetY := float64(int(float64(r.H)*.43)-int(land*float64(r.W)/65)) * 80 / float64(r.W)
		if flight >= 0 && flight < 5.5 {
			p := flight / 5.5
			x, y := broodArc(startX, targetX, targetY, p)
			// A segmented vapor/tendril trail follows the actual arc, not a straight
			// contrail. It stays short enough to read as a launched organic payload.
			for j := 1; j <= 8; j++ {
				q := p - float64(j)*.012
				if q < 0 {
					continue
				}
				tx, ty := broodArc(startX, targetX, targetY, q)
				r.idleGlow(tx, ty, 1.2, 0x70dfb0, (1-float64(j)/9)*.55)
			}
			bx, by := broodArc(startX, targetX, targetY, math.Max(0, p-.025))
			r.idleLine(bx, by, x, y, 0xb68b9b)
			r.idleLine(x, y, x, y, 0xe2dfaf)
			// Chitin hooks around a living warhead rather than metal rocket fins.
			r.idleLine(bx-.8, by+.5, bx+.8, by-.5, 0x88617c)
			if flight < .65 {
				r.idleGlow(startX, 21, 3, Green, .6*(1-flight/.65))
			}
		}
		r.broodImpact(targetX, targetY, flight-5.5)
	}
}

// Cubic trajectory leaves the sac nearly vertically, crests above the title,
// then dives into the far shore. Endpoint derivatives preserve the launch arc.
func broodArc(startX, targetX, targetY, p float64) (float64, float64) {
	q := 1 - p
	x := q*q*q*startX + 3*q*q*p*startX + 3*q*p*p*(targetX+7) + p*p*p*targetX
	y := q*q*q*21 + 3*q*q*p*(-11) + 3*q*p*p*(-10) + p*p*p*targetY
	return x, y
}

func (r *Raster) broodImpact(x, y, age float64) {
	if age < 0 || age > 7 {
		return
	}
	fade := 1 - smooth((age-3)/4)
	rise := smooth(age / .65)
	// A spreading ground flash and rising, lobed bio-plasma fireball. The
	// effect stays attached to the shore and is occluded by carrier and title.
	r.idleGlow(x, y, 7*rise+1, Green, .9*math.Max(0, 1-age/1.6))
	for i := 0; i < 5; i++ {
		dx := float64(i-2) * 1.7 * rise
		yy := y - (3+3*math.Sin(float64(i+1)*.55))*rise - age*.25
		r.idleGlow(x+dx, yy, 3.2, 0x83ce91, .8*fade)
		if age < 2 {
			r.idleGlow(x+dx*.6, yy+1, 2, 0xffefb6, (1-age/2)*.8)
		}
	}
	for i := 0; i < 18; i++ {
		life := 1 + hash(i, 591)*2.3
		if age > life {
			continue
		}
		dx := (hash(i, 592) - .5) * 9 * age
		dy := -hash(i, 593)*8*age + 2*age*age
		if dy > 0 {
			continue
		} // Debris lands at the distant shore, not in the foreground sea.
		r.idleLine(x+dx, y+dy, x+dx, y+dy, mix(Gold, 0x5a7a60, age/life))
	}
	for i := 0; i < 4; i++ {
		r.idleGlow(x+float64(i-2)*1.7+age*.45, y-4-age*.65, 3+age*.2, 0x44414e, .45*fade*smooth(age))
	}
}
