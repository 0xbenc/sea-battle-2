package visual

import "math"

func bombingAltitude(x float64) float64 {
	// One broad arc across the scene, with no flat middle or abrupt pull-up.
	// The shallow shoulders clear the subtitle at both resolutions.
	p := clamp((84-x)/88, 0, 1)
	return 12 + 6.5*math.Pow(math.Sin(math.Pi*p), 2)
}

// The same sortie returns at a much smaller scale, beyond the carrier's air
// traffic. Bomb release positions match the moving aircraft; impacts stay on
// the native shoreline at both resolutions. This replaces the entire dogfight.
func (r *Raster) idleBombingRun(age float64) {
	const startX, speed = 84.0, 9.2
	x := startX - (age-22)*speed
	altitude := bombingAltitude(x)
	if x >= -2 && x <= 82 {
		r.idleLine(x-1, bombingAltitude(x-1), x+1, bombingAltitude(x+1), 0xa9b8b8)
		r.idleLine(x, altitude-1, x, altitude, 0x738b97)
	}
	for i := 0; i < 5; i++ {
		targetX := 19 - float64(i)*3
		releaseX := targetX + 2
		released := 22 + (startX-releaseX)/speed
		fall := age - released
		const fallTime = 1.15
		px := targetX * float64(r.W) / 80
		land := math.Max(0, math.Sin(px*.08)+.5*math.Sin(px*.21))
		shoreY := float64(int(float64(r.H)*.43)-int(land*float64(r.W)/65)) * 80 / float64(r.W)
		if fall >= 0 && fall < fallTime {
			p := fall / fallTime
			bx, by := lerp(releaseX, targetX, p), lerp(bombingAltitude(releaseX)+1, shoreY, p*p)
			r.idleLine(bx, by, bx, by, Gold)
		}
		impact := fall - fallTime
		r.idleBurst(targetX, shoreY, impact, false)
		if impact >= 0 && impact < 4.5 {
			// Low, lingering dust connects the individual detonations into a line.
			r.idleGlow(targetX+impact*.3, shoreY-impact*.4, 1.8, 0x776655, .28*(1-impact/4.5))
		}
	}
}
