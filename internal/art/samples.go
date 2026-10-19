package art

// Sample exposes authored coverage before terminal compositing. Zero palette
// indices mean no contribution. A host owns layering, clipping and masks.
type Sample struct {
	X, Y   int
	Glyph  string // empty for a half-pixel sample
	FG, BG uint32
	Span   int // Unicode width; half samples have span 1
}

// Samples skips transparent locations and Unicode continuation cells. Coordinates
// remain local logical pixels (half) or terminal columns/rows (unicode).
func (c *Canvas) Samples(emit func(Sample)) {
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			n := c.Pixels[y*c.W+x]
			if n == 0 {
				continue
			}
			if c.Mode == "half" {
				emit(Sample{X: x, Y: y, FG: n, Span: 1})
				continue
			}
			t := c.Tiles[n-1]
			if t.FG == 0 && t.BG == 0 {
				continue
			}
			w, _ := GlyphWidth(t.Glyph)
			emit(Sample{X: x, Y: y, Glyph: t.Glyph, FG: t.FG, BG: t.BG, Span: w})
		}
	}
}
