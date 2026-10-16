package visual

import "math"

// Physical panel surfaces are composited last, keeping instruments readable.
func (r *Raster) console(x, row, w, h int, alien bool) {
	y := row * 2
	hh := h * 2
	base := RGB(0x152931)
	if alien {
		base = 0x2b2530
	}
	for yy := 0; yy < hh; yy++ {
		for xx := 0; xx < w; xx++ {
			c := mix(base, Ink, float64(yy)/float64(hh)*.28)
			if alien {
				v := math.Sin(float64(xx)*.16 + float64(yy)*.45)
				if v > .65 {
					c = mix(c, 0x624451, (v-.65)*.23)
				}
			} else if xx%27 == 0 {
				c = mix(c, Ink, .3)
			}
			r.put(x+xx, y+yy, c)
		}
	}
	edge := RGB(0x53686b)
	if alien {
		edge = 0x775769
	}
	r.line(x, y, x+w-1, y, edge)
	r.line(x, y+hh-1, x+w-1, y+hh-1, Ink)
	if alien {
		for xx := x + 1; xx < x+w-2; xx += 7 {
			r.line(xx, y, xx+2, y-2, 0x886775)
			r.line(xx+2, y-2, xx+5, y, 0x5e4355)
		}
		// Short luminous channels embedded in chitin, not a generic progress bar.
		for _, xx := range []int{x + 1, x + w - 2} {
			for yy := y + 3; yy < y+hh-1; yy++ {
				r.put(xx, yy, 0x3d4e4d)
				if yy%4 == 0 {
					r.put(xx, yy, 0x79b8a0)
				}
			}
		}
	} else {
		for _, xx := range []int{x + 1, x + w - 2} {
			r.put(xx, y+2, 0x93a4a0)
			r.put(xx, y+hh-3, 0x93a4a0)
		}
		r.line(x, y+1, x, y+hh-2, 0x334a51)
		r.line(x+w-1, y+1, x+w-1, y+hh-2, 0x334a51)
	}
}

func (r *Raster) instrumentBank(x, row int, t float64, alien bool) {
	if !alien {
		r.text(x, row-1, "HYDRAULIC MANIFOLD", Muted)
		for j := 0; j < 3; j++ {
			xx := x + j*10
			r.rect(xx, row*2, 5, 7, 0x09181c)
			r.line(xx, row*2, xx, row*2+7, 0x526b6e)
			r.line(xx+4, row*2, xx+4, row*2+7, 0x526b6e)
			level := int(2 + math.Sin(t*.8+float64(j))*1.4)
			r.rect(xx+1, row*2+level, 3, 6-level, 0x9e9066)
			r.line(xx, row*2+7, xx+4, row*2+7, Gold)
		}
	} else {
		r.text(x, row-1, "NEURAL ACTIVITY", Muted)
		r.rect(x, row*2, 31, 8, 0x101c24)
		for j := 0; j < 30; j++ {
			phase := float64(j)*.4 - t*2
			v := math.Sin(phase) * math.Exp(-math.Pow(math.Mod(float64(j)-t*8+200, 30)-15, 2)/24)
			r.put(x+j, row*2+4+int(v*3), Green)
		}
		r.line(x, row*2+8, x+30, row*2+8, 0x73566a)
	}
}
