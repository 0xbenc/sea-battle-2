package visual

import (
	"math"
	"sea-battle-2/internal/art"
	"sea-battle-2/internal/assets"
)

type colors struct{ sky, skyLow, sea, deep, crest, sun RGB }

var environments = []colors{
	{0x142b3d, 0x76999b, 0x1d525f, 0x092936, 0x54888b, 0xead8a4},
	{0x060f22, 0x283c56, 0x102a40, 0x061723, 0x3c637b, 0xa9c9cf},
	{0x201d2b, 0xaa7563, 0x243f49, 0x101e2d, 0x66828c, 0xffcd8a},
	{0x101b25, 0x526d62, 0x1f4845, 0x091e27, 0x527e71, 0xb7d791},
}

func (r *Raster) ocean(t float64, weather int, top bool) {
	p := environments[weather]
	horizon := int(float64(r.H) * .43)
	if top {
		horizon = -1
	}
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			var c RGB
			if y < horizon {
				d := float64(y) / float64(horizon)
				c = mix(p.sky, p.skyLow, d*d)
				// Broad cloud ribbons move continuously. No per-frame random noise.
				cloud := math.Sin(float64(x)*.047+float64(y)*.24+t*.025) + .45*math.Sin(float64(x)*.13-float64(y)*.4)
				if cloud > .83 {
					c = mix(c, p.sky, .24*smooth((cloud-.83)*3))
				}
			} else {
				d := float64(y-horizon) / float64(r.H-horizon)
				c = mix(p.sea, p.deep, d*.85)
				scale := .1 + d*.22
				wave := math.Sin(float64(y)*1.35+math.Sin(float64(x)*.04+t*.15)*1.6-t*1.3) + .45*math.Sin(float64(x)*scale-float64(y)*.32+t*.8)
				if wave > 1.0 {
					c = mix(c, p.crest, (wave-1)*.52)
				}
				if wave < -.9 {
					c = mix(c, p.deep, (-wave-.9)*.35)
				}
				// Broken light on the sea: narrow, offset reflection instead of full-width stripes.
				if !top {
					sunx := float64(r.W) * .73
					dist := math.Abs(float64(x) - sunx - math.Sin(float64(y)*1.8)*d*9)
					if dist < 3+d*14 && wave > .8 {
						c = mix(c, p.sun, (1-dist/(3+d*14))*.33)
					}
				}
			}
			r.put(x, y, c)
		}
	}
	if !top {
		sx, sy := int(float64(r.W)*.73), horizon-6
		radius := max(2, r.W/27)
		r.ellipse(sx, sy, radius, radius, p.sun, true)
		// Horizon haze bisects the sun and distant volcanic land.
		for y := sy; y < horizon; y++ {
			if y%3 == 0 {
				for x := sx - radius; x <= sx+radius; x++ {
					r.blend(x, y, p.skyLow, .55)
				}
			}
		}
		for x := 0; x < r.W; x++ {
			a := math.Sin(float64(x)*.08) + .5*math.Sin(float64(x)*.21)
			ht := int(math.Max(0, a) * float64(r.W) / 65)
			r.line(x, horizon-ht, x, horizon, mix(p.sky, p.sea, .35))
		}
	}
}
func (r *Raster) rain(t float64, weather int) {
	if weather != 2 {
		return
	}
	for i := 0; i < r.W/3; i++ {
		x := int(math.Mod(hash(i, 8)*float64(r.W)+t*13, float64(r.W)))
		y := int(math.Mod(hash(i, 11)*float64(r.H)+t*46, float64(r.H)))
		r.blend(x, y, 0x9da9b8, .3)
		r.blend(x-1, y+1, 0x9da9b8, .23)
	}
}
func (r *Raster) wake(x, y, w int, t float64, c RGB) {
	for side := -1; side <= 1; side += 2 {
		for j := 0; j < 5; j++ {
			xx := x + side*(w/2+j*2)
			yy := y + j/2
			v := .28 + .18*math.Sin(t*2+float64(j))
			r.blend(xx, yy, c, v)
			r.blend(xx+side, yy, c, v)
		}
	}
	for j := 0; j < 3; j++ {
		r.line(x-w/2+j*3, y+j, x+w/2-j*3, y+j, mix(environments[0].deep, c, .2-float64(j)*.04))
	}
}
func (r *Raster) reflection(c *art.Canvas, x, y int, t float64) {
	for yy := 0; yy < min(c.H/3, 9); yy++ {
		sy := c.H - 1 - yy
		offset := int(math.Sin(float64(yy)*1.8+t*2) * 2)
		for xx := 0; xx < c.W; xx++ {
			n := c.Pixels[sy*c.W+xx]
			if n > 0 {
				r.blend(x+xx+offset, y+yy, RGB(assets.Palette[n]), .18*(1-float64(yy)/10))
			}
		}
	}
}
func (r *Raster) smoke(x, y int, age float64, alien bool, scale float64) {
	for i := 0; i < 14; i++ {
		life := math.Mod(age*.65+hash(i, 44), 1)
		xx := x + int((math.Sin(float64(i)*2.4)+life*3)*scale)
		yy := y - int(life*25*scale)
		rad := max(1, int((1+life*3)*scale))
		c := RGB(0x565654)
		if alien {
			c = 0x657951
		}
		r.ellipse(xx, yy, rad, rad, mix(c, 0x253740, life*.55), true)
	}
}
func (r *Raster) fire(x, y int, t float64, alien bool, scale float64) {
	for i := 0; i < 11; i++ {
		p := math.Mod(t*1.6+hash(i, 72), 1)
		xx := x + int((hash(i, 12)-.5)*10*scale)
		yy := y - int(p*13*scale)
		c := mix(0xffe6a2, 0xd25d33, p)
		if alien {
			c = mix(0xcfffad, 0x4fba91, p)
		}
		r.ellipse(xx, yy, max(1, int((1-p)*3*scale)), max(1, int((1-p)*5*scale)), c, true)
	}
}
func (r *Raster) explosion(x, y int, age float64, alien bool, scale float64) {
	if age < 0 {
		return
	}
	if age < .18 {
		r.ellipse(x, y, int((3+age*70)*scale), int((2+age*40)*scale), Bone, true)
	}
	for i := 0; i < 42; i++ {
		a := hash(i, 3) * math.Pi * 2
		v := (5 + hash(i, 19)*17) * scale
		px := float64(x) + math.Cos(a)*v*age
		py := float64(y) + math.Sin(a)*v*age + 9*scale*age*age
		life := 1.1 + hash(i, 10)*1.3
		if age < life {
			c := mix(0xfff2bb, 0xa34335, age/life)
			if alien {
				c = mix(0xd8ff9e, 0x568459, age/life)
			}
			r.line(int(px), int(py), int(px-math.Cos(a)*2*scale), int(py-math.Sin(a)*2*scale), c)
		}
	}
	if age < 2.8 {
		r.smoke(x, y, age, alien, scale)
	}
}
func (r *Raster) splash(x, y int, age float64, scale float64) {
	if age < 0 || age > 4 {
		return
	}
	for i := 0; i < 38; i++ {
		v := (7 + hash(i, 4)*16) * scale
		dx := (hash(i, 6) - .5) * 12 * scale
		xx := float64(x) + dx*age
		yy := float64(y) - v*age + 12*scale*age*age
		if yy <= float64(y)+3*scale {
			r.line(int(xx), int(yy), int(xx), int(yy+2*scale), mix(Bone, 0x52858a, age/3))
		}
	}
	rx := int((3 + age*11) * scale)
	ry := max(1, int((1+age*2)*scale))
	for a := 0; a < 360; a += 12 {
		rad := float64(a) * math.Pi / 180
		r.blend(x+int(math.Cos(rad)*float64(rx)), y+int(math.Sin(rad)*float64(ry)), 0xa4c8c2, .65*(1-age/4))
	}
}
