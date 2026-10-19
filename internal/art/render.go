package art

import (
	"fmt"
	"github.com/rivo/uniseg"
	"strconv"
	"strings"
)

type Cell struct {
	Glyph, FG, BG string
	Span          int
}
type Frame struct {
	W, H  int
	Cells []Cell
}

func NewFrame(w, h int, bg string) *Frame {
	f := &Frame{W: w, H: h, Cells: make([]Cell, w*h)}
	for i := range f.Cells {
		f.Cells[i] = Cell{" ", "#F2F0E5", bg, 1}
	}
	return f
}
func (f *Frame) Put(x, y int, g, fg, bg string) {
	if x < 0 || y < 0 || x >= f.W || y >= f.H {
		return
	}
	w := uniseg.StringWidth(g)
	if w < 1 || w > 2 || x+w > f.W {
		return
	}
	i := y*f.W + x
	// Erase either half of a previously painted wide character.
	if f.Cells[i].Span == 0 && x > 0 {
		f.Cells[i-1] = Cell{" ", fg, bg, 1}
	}
	if f.Cells[i].Span == 2 && x+1 < f.W {
		f.Cells[i+1] = Cell{" ", fg, bg, 1}
	}
	if w == 2 && f.Cells[i+1].Span == 2 && x+2 < f.W {
		f.Cells[i+2] = Cell{" ", fg, bg, 1}
	}
	f.Cells[i] = Cell{g, fg, bg, w}
	if w == 2 {
		f.Cells[i+1] = Cell{"", fg, bg, 0}
	}
}
func (f *Frame) Text(x, y int, s, fg, bg string) {
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		g := gr.Str()
		w := uniseg.StringWidth(g)
		f.Put(x, y, g, fg, bg)
		x += w
		if x >= f.W {
			break
		}
	}
}
func (f *Frame) Fill(x, y, w, h int, bg string) {
	for yy := max(0, y); yy < min(f.H, y+h); yy++ {
		for xx := max(0, x); xx < min(f.W, x+w); xx++ {
			f.Put(xx, yy, " ", "#F2F0E5", bg)
		}
	}
}
func (f *Frame) Blit(src *Frame, x, y int) {
	for yy := 0; yy < src.H; yy++ {
		for xx := 0; xx < src.W; xx++ {
			c := src.Cells[yy*src.W+xx]
			if c.Span > 0 {
				f.Put(x+xx, y+yy, c.Glyph, c.FG, c.BG)
			}
		}
	}
}

// RenderRegion clips in terminal columns/rows, never splits a wide glyph.
func RenderRegion(d *Document, c *Canvas, bg string, ox, oy, w, h int) *Frame {
	f := NewFrame(w, h, bg)
	for y := 0; y < h; y++ {
		sy := y + oy
		if sy < 0 || sy >= c.TermHeight() {
			continue
		}
		for x := 0; x < w; x++ {
			sx := x + ox
			if sx < 0 || sx >= c.W {
				continue
			}
			if c.Mode == "half" {
				top := d.Color(c.Pixels[sy*2*c.W+sx], bg)
				bottom := bg
				if sy*2+1 < c.H {
					bottom = d.Color(c.Pixels[(sy*2+1)*c.W+sx], bg)
				}
				f.Put(x, y, "▀", top, bottom)
			} else if !c.Continuation(sx, sy) {
				t := c.TileAt(sx, sy)
				span := uniseg.StringWidth(t.Glyph)
				if x+span <= w {
					f.Put(x, y, t.Glyph, d.Color(t.FG, bg), d.Color(t.BG, bg))
				}
			}
		}
	}
	return f
}
func Render(d *Document, c *Canvas, bg string) *Frame {
	return RenderRegion(d, c, bg, 0, 0, c.W, c.TermHeight())
}
func (f *Frame) Plain() string {
	var b strings.Builder
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			b.WriteString(f.Cells[y*f.W+x].Glyph)
		}
		if y+1 < f.H {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
func (f *Frame) ANSI() string {
	var b strings.Builder
	cache := map[string]string{}
	color := func(s string, fg bool) string {
		key := s
		if fg {
			key += "f"
		}
		if v, ok := cache[key]; ok {
			return v
		}
		n, _ := strconv.ParseUint(strings.TrimPrefix(s, "#"), 16, 24)
		code := 48
		if fg {
			code = 38
		}
		v := fmt.Sprintf("\x1b[%d;2;%d;%d;%dm", code, n>>16, (n>>8)&255, n&255)
		cache[key] = v
		return v
	}
	for y := 0; y < f.H; y++ {
		fg, bg := "", ""
		for x := 0; x < f.W; x++ {
			c := f.Cells[y*f.W+x]
			if c.Span == 0 {
				continue
			}
			if c.FG != fg {
				b.WriteString(color(c.FG, true))
				fg = c.FG
			}
			if c.BG != bg {
				b.WriteString(color(c.BG, false))
				bg = c.BG
			}
			b.WriteString(c.Glyph)
		}
		b.WriteString("\x1b[0m")
		if y+1 < f.H {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
