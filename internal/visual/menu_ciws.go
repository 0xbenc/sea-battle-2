package visual

import "math"

const ciwsKill = 6.3

func ciwsDronePath(age float64) flightPoint {
	const turnRate = 2 * math.Pi / (ciwsKill - 2)
	if age < 2 {
		// Join the sweeping turn at full speed, rather than easing to a hover.
		p := age / 2
		h00, h10 := 2*p*p*p-3*p*p+1, p*p*p-2*p*p+p
		h01, h11 := -2*p*p*p+3*p*p, p*p*p-p*p
		return flightPoint{h00*89 + h10*(-14) + h01*70 + h11*(-18*turnRate), h00*12 + h10*6 + h01*22}
	}
	a := (age - 2) * turnRate
	return flightPoint{70 - 9*math.Sin(a), 17.5 + 4.5*math.Cos(a)}
}

func (r *Raster) ciwsDrone(p flightPoint, age float64) {
	// Orient the chitin manta along its actual tangent through the whole turn.
	pose := flightAt(ciwsDronePath, age, 0)
	speed := math.Hypot(pose.dx, pose.dy)
	ux, uy := -pose.dx/speed, -pose.dy/speed
	point := func(x, y float64) flightPoint {
		return flightPoint{p.x + x*ux - y*uy, p.y + x*uy + y*ux}
	}
	line := func(x, y, xx, yy float64, color RGB) {
		a, b := point(x, y), point(xx, yy)
		r.idleLine(a.x, a.y, b.x, b.y, color)
	}
	for _, side := range []float64{-1, 1} {
		wing := side * (2 + .35*math.Sin(age*5))
		line(2, 0, 4, wing, 0x805575)
		line(4, wing, -2, wing-side, 0xb493a3)
		line(-2, wing-side, -3, side*.7, 0xe0cda8)
	}
	for _, offset := range []float64{-.5, 0, .5} {
		line(-1, offset, 2, offset, 0x51394f)
	}
	line(0, -1, 2, -1, 0x9a748b)
	line(-2, 0, 0, 0, Green)
	nose := point(-2, 0)
	r.idleGlow(nose.x, nose.y, 1.5, Green, .3+.25*math.Sin(age*12))
	for _, hit := range []float64{5.25, 5.8} {
		tail := point(2, 0)
		r.idleBurst(tail.x, tail.y, age-hit, true)
	}
}

func (r *Raster) idleCIWS(age float64, foreground bool) {
	if age < 0 || age >= 17 {
		return
	}
	target := ciwsDronePath(ciwsKill)
	if !foreground {
		if age < ciwsKill {
			r.ciwsDrone(ciwsDronePath(age), age)
		}
		fall := age - ciwsKill
		r.idleBurst(target.x, target.y, fall, true)
		if fall >= 0 && fall < 1.5 {
			// The blast scatters dismembered wings outboard into the sea.
			for i := 0; i < 5; i++ {
				x := target.x + fall*(2+float64(i)*.8)
				y := target.y - float64(i%3)*fall + 4*fall*fall
				if y < 27 {
					r.idleLine(x, y, x+math.Cos(fall*12+float64(i))*2, y+math.Sin(fall*12+float64(i)), 0x98758b)
					r.idleGlow(x, y, 1.5, Green, .4*(1-fall/1.5))
				}
			}
		}
		r.idleWaterBurst(target.x+5, 27, fall-1.3)
		r.idleGlow(target.x+5, 27, 3, Green, math.Max(0, 1-math.Abs(fall-1.3)*3)*.5)
		return
	}
	// Split hatch, hydraulic snap-up, tracking beat, then three tightening bursts.
	rise := smooth((age-1.3)/.7) * (1 - smooth((age-11)/1.5))
	if rise <= 0 {
		return
	}
	r.idleRect(57, 28, 7, 1, 0x101e28)
	r.idleLine(57-rise*2, 28, 58-rise*2, 28, Bone)
	r.idleLine(63+rise*2, 28, 64+rise*2, 28, Bone)
	y := 28 - rise*4
	r.idleRect(59, y, 3, 28-y, 0x647e88)
	r.idleRect(58, y-1, 5, 2, 0xa7b7b4)
	r.idleLine(59, y-2, 61, y-2, Bone) // radar dome
	r.idleLine(58, y, 58, y+1, 0x364b59)
	aim := ciwsDronePath(math.Min(age+.18, ciwsKill))
	dx, dy := aim.x-61, aim.y-y
	length := math.Hypot(dx, dy)
	dx, dy = dx/length, dy/length
	firing := (age >= 3.1 && age < 3.65) || (age >= 4.5 && age < 5.25) || (age >= 5.65 && age < ciwsKill+.12)
	recoil := 0.0
	if firing {
		recoil = .5 * math.Abs(math.Sin(age*65))
	}
	muzzle := flightPoint{61 + dx*(4-recoil), y + dy*(4-recoil)}
	for _, offset := range []float64{-.5, .5} {
		r.idleLine(61-dy*offset, y+dx*offset, muzzle.x-dy*offset, muzzle.y+dx*offset, 0xbec9bb)
	}
	if firing {
		r.idleGlow(muzzle.x, muzzle.y, 2.2, Gold, .85)
		r.idleLine(muzzle.x, muzzle.y, muzzle.x+dx*1.5, muzzle.y+dy*1.5, Bone)
		for i := 0; i < 4; i++ {
			p := math.Mod(age*7+float64(i)*.25, 1)
			spread := math.Sin(float64(i)*4+age*21) * (1 - smooth((age-4.5)/1.8)) * 2
			x, yy := lerp(muzzle.x, aim.x+spread, p), lerp(muzzle.y, aim.y+spread, p)
			r.idleLine(x, yy, x+dx*1.6, yy+dy*1.6, Gold)
		}
		// Ejected brass tumbles onto the deck below the mount.
		for i := 0; i < 3; i++ {
			p := math.Mod(age*3+float64(i)/3, 1)
			r.idleLine(58-p*2, y+p*p*4, 58-p*2, y+p*p*4, 0xb9a26a)
		}
	}
	if age > ciwsKill && age < 10 {
		p := (age - ciwsKill) / (10 - ciwsKill)
		r.idleGlow(muzzle.x+p*2, muzzle.y-p*2, 1.5+p, 0x8d9696, .3*(1-p))
	}
}
