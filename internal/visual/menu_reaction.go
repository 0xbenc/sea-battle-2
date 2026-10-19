package visual

import (
	"math"
	"sea-battle-2/internal/assets"
)

// Coordinates are SD half-pixels, so a resize preserves the point in the scene.
type MenuReaction struct {
	Kind      string
	X, Y, Age float64
}

type menuMasks struct{ title, ship, events *Raster }

func newMenuMasks(sd bool, idle MenuIdle) menuMasks {
	w, h, size, row := 160, 48, "HD", 5
	if sd {
		w, h, size, row = 80, 24, "SD", 3
	}
	m := menuMasks{New(w, h), New(w, h), New(w, h)}
	m.title.bigWord("SEA BATTLE II", row, Bone)
	c := assets.Get("ship-4-side", size)
	m.ship.asset(c, (w-c.W)/2, m.ship.H*72/100-c.Anchor.Y, 0, 0, false)
	m.events.menuBackground(idle)
	m.events.menuForeground(idle)
	return m
}
func (m menuMasks) water(x, y int) bool {
	r := m.ship
	if x < 0 || x >= r.W || y <= int(float64(r.H)*.43) || y >= r.H {
		return false
	}
	i := y*r.W + x
	return r.Pixels[i] == 0 && m.events.Pixels[i] == 0
}

// Hit testing follows actual painted pixels, including moving story objects.
// A terminal click covers two half-pixels; prefer an occupied pixel over water.
func MenuReactionAt(sd bool, idle MenuIdle, x, row int) MenuReaction {
	masks := newMenuMasks(sd, idle)
	w, h := masks.ship.W, masks.ship.H/2
	if x < 0 || x >= w || row < 0 || row >= h {
		return MenuReaction{}
	}
	scale := float64(w) / 80
	reaction := func(kind string, y int) MenuReaction {
		return MenuReaction{Kind: kind, X: float64(x) / scale, Y: float64(y) / scale}
	}
	panelW, panelH, subtitle := 108, 7, 12
	if sd {
		panelW, panelH, subtitle = 72, 5, 7
	}
	if row >= h-panelH-1 && row < h-1 && x >= (w-panelW)/2 && x < (w+panelW)/2 {
		return MenuReaction{}
	}
	if row == subtitle && x >= (w-len("OPERATION: UNEXPECTED GUESTS"))/2 && x < (w+len("OPERATION: UNEXPECTED GUESTS"))/2 {
		return MenuReaction{}
	}
	for y := row * 2; y < row*2+2; y++ {
		if masks.title.Pixels[y*w+x] != 0 {
			return reaction("glint", y)
		}
	}
	for y := row * 2; y < row*2+2; y++ {
		if masks.events.Pixels[y*w+x] != 0 {
			return MenuReaction{}
		}
	}
	for y := row * 2; y < row*2+2; y++ {
		if masks.ship.Pixels[y*w+x] != 0 {
			return reaction("spark", y)
		}
	}
	if masks.water(x, row*2) && masks.water(x, row*2+1) {
		return reaction("ripple", row*2+1)
	}
	return MenuReaction{}
}

func (r *Raster) menuReaction(fx MenuReaction, idle MenuIdle) {
	if fx.Kind == "" || fx.Age < 0 || fx.Age > .8 {
		return
	}
	masks := newMenuMasks(r.W <= 80, idle)
	scale := float64(r.W) / 80
	switch fx.Kind {
	case "ripple":
		if fx.Age > .65 {
			return
		}
		p := fx.Age / .65
		// A low elliptical ring, fading before it grows into a distracting wake.
		painted := make(map[int]bool, 40)
		for i := 0; i < 40; i++ {
			angle := float64(i) * 2 * math.Pi / 40
			x := int(math.Round((fx.X + (.5+3.5*p)*math.Cos(angle)) * scale))
			y := int(math.Round((fx.Y + (.3+1.1*p)*math.Sin(angle)) * scale))
			index := y*r.W + x
			if masks.water(x, y) && !painted[index] {
				r.blend(x, y, 0xa0c2c1, .55*(1-p))
				painted[index] = true
			}
		}
	case "spark":
		if fx.Age > .4 {
			return
		}
		p := fx.Age / .4
		for i := 0; i < 5; i++ {
			a := float64(i)*2*math.Pi/5 + .4
			x, y := fx.X+math.Cos(a)*2*p, fx.Y+math.Sin(a)*2*p+p*p
			r.idleLine(x, y, x, y, mix(Gold, 0x62747b, p))
		}
		r.idleGlow(fx.X, fx.Y, 1.3, Bone, .65*(1-p))
	case "glint":
		p := fx.Age / .8
		for y := 0; y < r.H; y++ {
			for x := 0; x < r.W; x++ {
				if masks.title.Pixels[y*r.W+x] == 0 {
					continue
				}
				d := math.Abs(float64(x)/scale - fx.X + (float64(y)/scale-fx.Y)*.5 - p*9)
				r.blend(x, y, 0xffffff, math.Max(0, 1-d/2.5)*math.Sin(math.Pi*p)*.95)
			}
		}
	}
}
