package visual

import "math"

func (r *Raster) idleRect(x, y, w, h float64, c RGB) {
	s := float64(r.W) / 80
	r.rect(int(math.Round(x*s)), int(math.Round(y*s)), max(1, int(w*s)), max(1, int(h*s)), c)
}
func (r *Raster) idleWaterBurst(x, y, age float64) {
	if age < 0 || age > 2 {
		return
	}
	for i := 0; i < 9; i++ {
		dx := (hash(i, 811) - .5) * 7 * age
		dy := -hash(i, 812)*6*age + 3*age*age
		if dy > 0 {
			continue
		}
		r.idleLine(x+dx, y+dy, x+dx, y+dy, mix(Bone, 0x345563, age/2))
	}
	r.idleLine(x-age*4, y, x+age*4, y, mix(0x6b9b9e, 0x243f49, age/2))
}

func (r *Raster) idleHelicopter(x, y, t float64) {
	r.idleRect(x-3, y-1, 6, 2, 0x7e9292)
	r.idleRect(x+1, y-1, 2, 1, 0xade3dc)
	r.idleLine(x-8, y-1, x-3, y, 0x7e9292)
	r.idleLine(x-8, y-2, x-8, y, Muted)
	r.idleLine(x, y-1, x, y-3, Muted)
	rotor := 3 + 4*math.Abs(math.Sin(t*31))
	r.idleLine(x-rotor, y-3, x+rotor, y-3, 0xb4c0b1)
	r.idleLine(x-3, y+2, x+3, y+2, 0x546e76)
	r.idleLine(x-2, y+1, x-2, y+2, Muted)
	r.idleLine(x+2, y+1, x+2, y+2, Muted)
}
func (r *Raster) idleCrate(x, y, legs, t float64) {
	// Beige shipping container, then a split lid and suspicious mint eyes.
	r.idleRect(x-2, y-1, 5, 3, 0x9c845a)
	r.idleLine(x-2, y-1-legs*.7, x+2, y-1-legs*.7, 0xc7b58a)
	r.idleLine(x, y-1, x, y+1, 0xd9cc9b)
	if legs > 0 {
		r.idleLine(x-1, y, x+1, y, 0x4b344f)
		r.idleLine(x-1, y, x-1, y, Green)
		r.idleLine(x+1, y, x+1, y, Green)
		for _, side := range []float64{-1, 1} {
			step := math.Sin(t*17+side) * .8
			r.idleLine(x+side, y+1, x+side*(3+step)*legs, y+2, 0x9d758a)
			r.idleLine(x+side*2, y+1, x+side*3*legs, y+2-step*.4, 0xb1a386)
		}
	}
}
func (r *Raster) idleResupply(a float64) {
	if a < 0 || a >= 28 {
		return
	}
	// Approach; hold a steady hover while the cable pays out; depart empty.
	if a < 13 {
		x, y := lerp(-12, 36, smooth(a/4)), 19.5
		if a > 9 {
			x = lerp(36, 92, smooth((a-9)/4))
			y -= smooth((a-9)/4) * 5
		}
		r.idleHelicopter(x, y, a)
		if a < 9 {
			cy := 23 + smooth((a-4)/4)*4
			if a < 4 {
				cy = 23
			}
			r.idleLine(x, y+1, x, cy-1, 0x829a9a)
			r.idleCrate(x, cy, 0, a)
		} else if a < 10 {
			r.idleLine(x, y+1, x, y+1+5*(10-a), 0x829a9a)
		}
	}
	if a >= 9 && a < 24 {
		x, y := 36.0, 27.0
		legs := smooth((a - 14.5) / 1.5)
		if a >= 17 && a < 22 {
			x = lerp(36, 69, smooth((a-17)/5))
			y += math.Sin(a*18) * .2
		}
		if a >= 22 {
			p := (a - 22) / 2
			x = lerp(69, 75, p)
			y = 27 - 4*math.Sin(p*math.Pi) + 6*p*p
		}
		r.idleCrate(x, y, legs, a)
	}
	r.idleWaterBurst(75, 33, a-24)
}

func flarePosition(release, age float64, index int) (float64, float64) {
	return release + age*1.7, 18 + float64(index-1)*age*1.8 + .15*age*age
}
func (r *Raster) idleFlareEscape(a float64) {
	if a < 0 || a >= 16 {
		return
	}
	x := 87 - a*8
	if x > -8 {
		r.idleJet(x, 18, -1, false, true)
		// A damaged engine leaves a broken charcoal trail, not a second jet.
		for i := 1; i <= 5; i++ {
			r.idleGlow(x+float64(i)*2, 18+math.Sin(a+float64(i))*.4, 1.4, 0x45414a, .45*(1-float64(i)/6))
		}
	}
	for i := 0; i < 3; i++ {
		if i == 1 && a >= 7 {
			continue
		} // The bitten flare is consumed.
		releaseTime := 4 + float64(i)*.3
		age := a - releaseTime
		if age < 0 || age > 5 {
			continue
		}
		release := 87 - releaseTime*8
		fx, fy := flarePosition(release, age, i)
		r.idleGlow(fx, fy, 2.5, Gold, .9*(1-age/5))
		r.idleLine(fx, fy, fx, fy, Bone)
		r.idleLine(fx-.8, fy-.5, fx, fy, Gold)
	}
	// The pursuer abandons the fighter and bites the middle flare.
	releaseTime := 4.3
	fx, fy := flarePosition(87-releaseTime*8, 7-releaseTime, 1)
	if a < 7 {
		ax, ay := x+13, 18.7
		if a >= 4.5 {
			p := smooth((a - 4.5) / 2.5)
			ax = lerp(64, fx, p)
			ay = lerp(18.7, fy, p) - math.Sin(p*math.Pi)*3
		}
		r.idlePlane(ax, ay, -1, true, false, a)
	}
	r.idleBurst(fx, fy, a-7, true)
}

func (r *Raster) idleDropship(x, y, bank float64) {
	// Beetle-like troop sac: violet plates, membrane outriggers, mint grafts.
	r.idleRect(x-3, y-1, 7, 3, 0x74506c)
	r.idleRect(x-1, y-2, 4, 1, 0xa47a8a)
	r.idleLine(x-5, y+bank, x-2, y, 0xc0b48d)
	r.idleLine(x+3, y, x+6, y-bank, 0x92718c)
	r.idleLine(x-3, y, x-2, y, Green)
	r.idleLine(x+1, y, x+2, y, Green)
	r.idleLine(x-1, y+2, x+3, y+2, 0x43364e)
}
func (r *Raster) idleBadLanding(a float64) {
	if a < 0 || a >= 17 {
		return
	}
	// Three water skips, each lower and less controlled, then the beach.
	xs := []float64{87, 64, 43, 25, 11}
	ys := []float64{11, 25, 24, 23, r.shoreHeight(11)}
	times := []float64{0, 3, 5, 7, 9}
	if a < 9 {
		leg := 0
		for leg < 3 && a >= times[leg+1] {
			leg++
		}
		p := (a - times[leg]) / (times[leg+1] - times[leg])
		x, y := lerp(xs[leg], xs[leg+1], p), lerp(ys[leg], ys[leg+1], p)
		if leg > 0 {
			y -= math.Sin(p*math.Pi) * (5 - float64(leg))
		}
		r.idleDropship(x, y, math.Sin(a*7)*1.5)
	}
	for i := 1; i <= 3; i++ {
		r.idleWaterBurst(xs[i], ys[i], a-times[i])
	}
	r.broodImpact(11, ys[4], a-9)
}
func (r *Raster) shoreHeight(x float64) float64 {
	px := x * float64(r.W) / 80
	land := math.Max(0, math.Sin(px*.08)+.5*math.Sin(px*.21))
	return float64(int(float64(r.H)*.43)-int(land*float64(r.W)/65)) * 80 / float64(r.W)
}

const awakeningHit = 8.5

func awakeningShipX(age float64) float64 {
	return 91 - 2.7*math.Min(age, awakeningHit)
}

func (r *Raster) awakeningShip(age float64) {
	x := awakeningShipX(age)
	damage := math.Max(0, age-awakeningHit)
	if damage > 8 {
		return
	}
	y := 23 + damage*.3
	// A smaller American escort crosses the midground behind the carrier.
	// Draw a filled hull at native pixel spacing so HD has no comb-like gaps.
	scale := float64(r.W) / 80
	tilt := damage * .055
	line := func(ax, ay, bx, by float64, color RGB) {
		r.idleLine(x+ax, y+ay-ax*tilt, x+bx, y+by-bx*tilt, color)
	}
	for i := int(-8 * scale); i <= int(6*scale); i++ {
		dx := float64(i) / scale
		depth := math.Min(1.8, (dx+8)*.6)
		line(dx, 0, dx, depth, 0x667f89)
		line(dx, 0, dx, 0, 0xb6c3b6)
	}
	for dy := -2.0; dy < 0; dy += 1 / scale {
		line(-2, dy, 3, dy, 0x879d9e)
	}
	line(-1, -3, 1, -3, 0xc5cbb7)
	line(0, -3, 0, -6, 0x849c9e)
	line(0, -5, 2, -5, Bone)
	line(1, -4, 2, -4, Rust)
	line(-5, -1, -3, -1, 0x9caeac)
	if damage == 0 {
		r.idleLine(x+7, y+1, x+11, y+1, 0x6b9b9e)
	} else {
		for i := 0; i < 3; i++ {
			fx := x - 3 + float64(i)*3
			r.idleGlow(fx, y-1, 2, Gold, .55*(1-damage/8))
			r.idleLine(fx, y-1, fx+math.Sin(age*9+float64(i)), y-2-math.Abs(math.Sin(age*7+float64(i)))*2, Rust)
			for j := 0; j < 3; j++ {
				p := math.Mod(damage*.5+float64(j)/3, 1)
				r.idleGlow(fx+p*3, y-2-p*7, 1.5+p*2, 0x30333b, (1-p)*.7)
			}
		}
	}
}

func (r *Raster) idleAwakening(a float64) {
	if a < 0 || a >= 18 {
		return
	}
	r.awakeningShip(a)
	const x = 12.0
	targetX := awakeningShipX(awakeningHit)
	const targetY = 22.0
	ground := r.shoreHeight(x)
	open := smooth(a/3) * (1 - smooth((a-11)/5))
	// A solid chitin mound rises out of the land before the jaws unfold.
	scale := r.W / 80
	for column := -5 * scale; column <= 5*scale; column++ {
		dx := float64(column) / float64(scale)
		height := math.Sqrt(math.Max(0, 1-dx*dx/25)) * 3 * open
		px := int((x + dx) * float64(scale))
		for py := int((ground - height) * float64(scale)); py <= int(ground*float64(scale)); py++ {
			r.blend(px, py, 0x67485f, open*.9)
		}
	}
	for _, side := range []float64{-1, 1} {
		r.idleLine(x+side*4, ground, x+side*(2+open*2), ground-open*5, 0x8d6c80)
		r.idleLine(x+side*(2+open*2), ground-open*5, x+side, ground-open*3, 0xb8ac88)
	}
	mouth := ground - 3*open
	r.idleRect(x-1, mouth, 3, 2, mix(0x352c40, Green, open*.75))
	if a > 3 && a < 6 {
		r.idleGlow(x, mouth, 2+(a-3)*.6, Green, (a-3)/3*.8)
	}
	if a >= 6 && a < 8.5 {
		p := (a - 6) / 2.5
		px, py := lerp(x, targetX, p), lerp(ground-3, targetY, p)-20*math.Sin(p*math.Pi)
		for i := 0; i < 6; i++ {
			q := math.Max(0, p-float64(i)*.015)
			tx, ty := lerp(x, targetX, q), lerp(ground-3, targetY, q)-20*math.Sin(q*math.Pi)
			r.idleGlow(tx, ty, 1.6, Green, (1-float64(i)/6)*.7)
		}
		r.idleLine(px, py, px, py, 0xdfffc3)
	}
	r.broodImpact(targetX, targetY, a-awakeningHit)
	r.idleBurst(targetX-2, targetY, a-awakeningHit-.12, false)
	r.idleBurst(targetX+3, targetY, a-awakeningHit-.5, false)
}
