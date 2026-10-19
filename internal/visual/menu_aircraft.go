package visual

import "math"

// Deliberate SD silhouettes: twin tails, swept wings, a bright canopy and a
// tapered nose. HD uses the same silhouette with extra surface highlights.
var menuJetNear = []string{
	" tt       ",
	"  t  cc   ",
	"  bbbbbbn>",
	" tt bww   ",
	"    ww    ",
}
var menuJetFar = []string{
	"t cc   ",
	" bbbbb>",
	" tww   ",
}

func (r *Raster) idleJet(x, y float64, direction int, near, engine bool) {
	r.jetAttitude(x, y, direction, near, engine, 0, 0)
}

func (r *Raster) jetAttitude(x, y float64, direction int, near, engine bool, pitch, bank float64) {
	sprite, ox, oy := menuJetFar, 3, 1
	if near {
		sprite, ox, oy = menuJetNear, 5, 2
	}
	scale := r.W / 80
	width := 1 - .65*math.Abs(bank)
	vertical := 1 - .6*math.Abs(bank)
	palette := map[rune]RGB{'t': 0x849eaa, 'c': 0xa9e6e4, 'b': 0xc5cdd0, 'n': Bone, '>': Bone, 'w': 0x536e82}
	for yy, row := range sprite {
		for xx, ch := range row {
			color, ok := palette[ch]
			if !ok {
				continue
			}
			px := int(math.Round((x + float64((xx-ox)*direction)*width) * float64(scale)))
			py := int(math.Round((y + float64(yy-oy)*vertical + float64(xx-ox)*math.Sin(pitch)) * float64(scale)))
			r.rect(px, py, scale, scale, color)
			if scale > 1 && ch == 'b' {
				r.put(px, py, 0xe2e3d2)
			}
		}
	}
	if engine {
		tail := x - float64((ox-1)*direction)*width
		ty := y - float64(ox-1)*math.Sin(pitch)
		r.idleLine(tail-float64(direction)*width, ty-math.Sin(pitch), tail-2*float64(direction)*width, ty-2*math.Sin(pitch), Gold)
	}
}
