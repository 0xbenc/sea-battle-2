// Package visual composes logical half pixels before encoding terminal cells.
package visual

import (
	"fmt"
	"git.occ.farm/occ/dotmason/art"
	"math"
	"sea-battle-2/internal/assets"
	"strings"
	"unicode/utf8"
)

type RGB uint32

const (
	Ink   RGB = 0x071319
	Panel RGB = 0x12262d
	Bone  RGB = 0xe5dfba
	Muted RGB = 0x8da8a8
	Cyan  RGB = 0x79d7d1
	Gold  RGB = 0xeac078
	Green RGB = 0x9deaac
	Rust  RGB = 0xe47f58
)

func mix(a, b RGB, t float64) RGB {
	t = clamp(t, 0, 1)
	return RGB(uint32(float64((a>>16)&255)*(1-t)+float64((b>>16)&255)*t)<<16 | uint32(float64((a>>8)&255)*(1-t)+float64((b>>8)&255)*t)<<8 | uint32(float64(a&255)*(1-t)+float64(b&255)*t))
}
func clamp(v, a, b float64) float64 { return math.Max(a, math.Min(b, v)) }
func lerp(a, b, t float64) float64  { return a + (b-a)*t }
func hash(x, y int) float64 {
	n := uint32(x)*374761393 + uint32(y)*668265263
	n = (n ^ (n >> 13)) * 1274126177
	return float64(n^(n>>16)) / 4294967295
}
func smooth(x float64) float64 { x = clamp(x, 0, 1); return x * x * (3 - 2*x) }

type Cell struct {
	Ch     string
	FG, BG RGB
}
type Frame struct {
	W, H  int
	Cells []Cell
}
type Raster struct {
	W, H   int
	Pixels []RGB
	Texts  []label
}
type label struct {
	x, y   int
	s      string
	fg, bg RGB
	solid  bool
}

func New(w, h int) *Raster { return &Raster{W: w, H: h * 2, Pixels: make([]RGB, w*h*2)} }
func (r *Raster) put(x, y int, c RGB) {
	if x >= 0 && y >= 0 && x < r.W && y < r.H {
		r.Pixels[y*r.W+x] = c
	}
}
func (r *Raster) blend(x, y int, c RGB, a float64) {
	if x >= 0 && y >= 0 && x < r.W && y < r.H {
		r.put(x, y, mix(r.Pixels[y*r.W+x], c, a))
	}
}
func (r *Raster) rect(x, y, w, h int, c RGB) {
	for yy := max(0, y); yy < min(r.H, y+h); yy++ {
		for xx := max(0, x); xx < min(r.W, x+w); xx++ {
			r.put(xx, yy, c)
		}
	}
}
func (r *Raster) line(x0, y0, x1, y1 int, c RGB) {
	dx := math.Abs(float64(x1 - x0))
	dy := math.Abs(float64(y1 - y0))
	n := int(math.Max(dx, dy))
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(max(1, n))
		r.put(int(math.Round(lerp(float64(x0), float64(x1), t))), int(math.Round(lerp(float64(y0), float64(y1), t))), c)
	}
}
func (r *Raster) ellipse(cx, cy, rx, ry int, c RGB, fill bool) {
	for y := -ry; y <= ry; y++ {
		for x := -rx; x <= rx; x++ {
			d := float64(x*x)/float64(max(1, rx*rx)) + float64(y*y)/float64(max(1, ry*ry))
			if d <= 1 && (fill || d > .65) {
				r.put(cx+x, cy+y, c)
			}
		}
	}
}
func (r *Raster) text(x, y int, s string, c RGB) {
	r.Texts = append(r.Texts, label{x: x, y: y, s: s, fg: c})
}
func (r *Raster) solidText(x, y int, s string, c, bg RGB) {
	r.Texts = append(r.Texts, label{x: x, y: y, s: s, fg: c, bg: bg, solid: true})
}
func (r *Raster) center(y int, s string, c RGB) { r.text((r.W-utf8.RuneCountInString(s))/2, y, s, c) }
func (r *Raster) plaque(x, y, w, h int, c RGB) {
	r.rect(x, y*2, w, h*2, c)
	r.line(x, y*2, x+w-1, y*2, mix(c, Bone, .25))
	r.line(x, y*2+h*2-1, x+w-1, y*2+h*2-1, Ink)
}
func (r *Raster) asset(c *art.Canvas, x, y int, tint RGB, amount float64, mirror bool) {
	c.Samples(func(p art.Sample) {
		xx := p.X
		if mirror {
			xx = c.W - 1 - xx
		}
		r.put(x+xx, y+p.Y, mix(RGB(assets.Palette[p.FG]), tint, amount))
	})
}
func (r *Raster) ship(i int, view, size string, x, y int, tint RGB, amount float64) {
	r.asset(assets.Get(fmt.Sprintf("ship-%d-%s", i, view), size), x, y, tint, amount, false)
}
func (r *Raster) frame() *Frame {
	f := &Frame{r.W, r.H / 2, make([]Cell, r.W*r.H/2)}
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			f.Cells[y*f.W+x] = Cell{"▀", r.Pixels[y*2*r.W+x], r.Pixels[(y*2+1)*r.W+x]}
		}
	}
	for _, l := range r.Texts {
		x := l.x
		for _, ch := range l.s {
			if x >= 0 && x < f.W && l.y >= 0 && l.y < f.H {
				c := &f.Cells[l.y*f.W+x]
				bg := c.BG
				if l.solid {
					bg = l.bg
				}
				*c = Cell{string(ch), l.fg, bg}
			}
			x++
		}
	}
	return f
}
func (f *Frame) Plain() string {
	var b strings.Builder
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			b.WriteString(f.Cells[y*f.W+x].Ch)
		}
		if y < f.H-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
func (f *Frame) ANSI() string {
	var b strings.Builder
	for y := 0; y < f.H; y++ {
		fg, bg := RGB(0xffffffff), RGB(0xffffffff)
		for x := 0; x < f.W; x++ {
			c := f.Cells[y*f.W+x]
			if c.FG != fg {
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm", c.FG>>16, (c.FG>>8)&255, c.FG&255)
				fg = c.FG
			}
			if c.BG != bg {
				fmt.Fprintf(&b, "\x1b[48;2;%d;%d;%dm", c.BG>>16, (c.BG>>8)&255, c.BG&255)
				bg = c.BG
			}
			b.WriteString(c.Ch)
		}
		b.WriteString("\x1b[0m")
		if y < f.H-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (r *Raster) fadingEllipse(cx, cy, rx, ry int, c RGB, opacity float64) {
	for y := -ry; y <= ry; y++ {
		for x := -rx; x <= rx; x++ {
			if float64(x*x)/float64(max(1, rx*rx))+float64(y*y)/float64(max(1, ry*ry)) <= 1 {
				r.blend(cx+x, cy+y, c, opacity)
			}
		}
	}
}
