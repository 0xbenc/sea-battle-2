package visual

import "math"

type flightPoint struct{ x, y float64 }
type flightPose struct {
	flightPoint
	dx, dy, bank float64
}

// A brief tension beat, then a hard accelerating deck stroke. Exit speed
// carries through the bow; the climb starts only as the jet clears the deck.
func launchPath(age float64) flightPoint {
	const hold, stroke, distance = .35, 1.05, 24.0
	t := math.Max(0, age-hold)
	x := 30 - distance*math.Pow(t/stroke, 2)
	if t > stroke {
		x = 30 - distance - (2*distance/stroke)*(t-stroke)
	}
	return flightPoint{x, 27 - 12*smooth((t-stroke+.12)/1.15)}
}

func flightAt(path func(float64) flightPoint, t, bank float64) flightPose {
	const step = .01
	p, before, after := path(t), path(t-step), path(t+step)
	return flightPose{p, (after.x - before.x) / (2 * step), (after.y - before.y) / (2 * step), bank}
}

// The pursuit joins a three-second banked reversal with matching position and
// velocity. The old instant left/right sprite flip had no physical turn at all.
func dogfightPath(age float64, high bool) flightPoint {
	const speed = 72.0 / 7
	if age < 29 {
		p := (age - 22) / 7
		height := 1.3
		if high {
			height = -12
		}
		return flightPoint{87 - 72*p, 18 + height*math.Pow(math.Sin(p*math.Pi), 2)}
	}
	if age < 32 {
		angle := (age - 29) / 3 * math.Pi
		return flightPoint{15 - speed*3/math.Pi*math.Sin(angle), 18 - 5.5*(1-math.Cos(angle))}
	}
	q := age - 32
	distance := speed*q + 2*q*q
	if q > 2 {
		distance = speed*2 + 8 + (speed+8)*(q-2)
	}
	return flightPoint{15 + distance, 7 + 5*smooth((age-33)/4)}
}
func pursuitPath(age float64, high bool) flightPoint {
	p := dogfightPath(age-.65, high)
	// The pursuer takes a wider line. The tighter human turn puts the alien
	// ahead of the nose on the exit, making the winning shot face forward.
	p.x += 5 + 12*smooth((age-29)/3)
	p.y += 1.4
	return p
}

const landingTouchdown = 44.4
const landingSpeed = 47.0 / 5.4
const landingArrestTime = 20.0 / landingSpeed

func landingPath(age float64) flightPoint {
	if age < landingTouchdown {
		p := (age - 39) / 5.4
		// Constant forward approach, progressively flattening descent into flare.
		return flightPoint{87 - 47*p, 21 + 5.5*(2*p-p*p)}
	}
	q := clamp(age-landingTouchdown, 0, landingArrestTime)
	// Arresting wire takes ten pixels to stop the aircraft, with continuous
	// touchdown speed. Suspension settles without a last-second vertical drop.
	return flightPoint{40 - landingSpeed*q + landingSpeed*q*q/(2*landingArrestTime), 26.5 + .5*smooth(q/.4)}
}
func (r *Raster) flyingJet(p flightPose, near, engine bool) {
	direction := 1
	if p.dx < 0 {
		direction = -1
	}
	pitch := math.Atan2(p.dy, math.Abs(p.dx))
	if near && !engine {
		pitch -= .08
	} // Slight nose-up approach attitude.
	r.jetAttitude(p.x, p.y, direction, near, engine, clamp(pitch, -.55, .55), p.bank)
}
func (r *Raster) landingJet(age float64) {
	p := landingPath(age)
	if age >= landingTouchdown {
		r.idleJet(p.x, p.y, -1, true, false)
	} else {
		r.flyingJet(flightAt(landingPath, age, 0), true, false)
	}
	if age >= 41.5 && age < landingTouchdown+.4 {
		for _, dx := range []float64{-2, 2} {
			r.idleLine(p.x+dx, p.y+1, p.x+dx, p.y+2, 0x82939a)
			r.idleLine(p.x+dx-.5, p.y+2, p.x+dx+.5, p.y+2, 0x273d47)
		}
	}
	contact := age - landingTouchdown
	if contact >= 0 && contact < .65 {
		r.idleGlow(p.x+2, 29, 2, Bone, .35*(1-contact/.65))
	}
	if contact >= 0 && contact < landingArrestTime {
		r.idleLine(43, 28.5, p.x+3, 28.5, mix(Muted, 0x3d535b, contact/landingArrestTime))
	}
}
