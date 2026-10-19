package art

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Frames are full images. No timestamp, loop mode or disposal behavior is stored.
func (c Canvas) Units() uint32 {
	if c.Hold == 0 {
		return 1
	}
	return c.Hold
}
func (t Track) TotalUnits() uint64 {
	var n uint64
	for _, f := range t.Frames {
		n += uint64(f.Units())
	}
	return n
}

// AtUnit is clock-free and non-looping. The host chooses whether to wrap or stop.
func (t Track) AtUnit(unit uint64) (*Canvas, bool) {
	for i := range t.Frames {
		n := uint64(t.Frames[i].Units())
		if unit < n {
			return &t.Frames[i], true
		}
		unit -= n
	}
	return nil, false
}
func (c *Canvas) Track(name string) (*Track, error) {
	for i := range c.Tracks {
		if c.Tracks[i].Name == name {
			return &c.Tracks[i], nil
		}
	}
	return nil, fmt.Errorf("unknown track %q", name)
}

// ResolvePalette never mutates the asset; fixed entries cannot be overridden.
func (d *Document) ResolvePalette(args map[string]string) ([]string, error) {
	p := append([]string(nil), d.Palette...)
	for k, v := range args {
		i, ok := d.Slots[k]
		if !ok {
			return nil, fmt.Errorf("unknown color slot %q", k)
		}
		if i == 0 || int(i) > len(p) {
			return nil, fmt.Errorf("invalid palette index for slot %q", k)
		}
		if !ValidColor(v) {
			return nil, fmt.Errorf("invalid color for slot %q", k)
		}
		p[i-1] = strings.ToUpper(v)
	}
	return p, nil
}
func (d *Document) validateAnimation() error {
	seen := map[uint32]bool{}
	for k, v := range d.Slots {
		if !ValidName(k) || v == 0 || int(v) > len(d.Palette) || seen[v] {
			return fmt.Errorf("slots need unique valid names and distinct nonzero palette entries")
		}
		seen[v] = true
	}
	for k, v := range d.Extensions {
		if !strings.Contains(k, ".") || !ValidName(k) || !json.Valid(v) {
			return fmt.Errorf("invalid namespaced extension %q", k)
		}
	}
	total, frames := 0, 0
	ids := map[string]bool{}
	checkID := func(id string) error {
		if id == "" {
			return nil
		}
		if !ValidName(id) || ids[id] {
			return fmt.Errorf("invalid or duplicate asset id %q", id)
		}
		ids[id] = true
		return nil
	}
	for _, ds := range d.Designs {
		if err := checkID(ds.ID); err != nil {
			return err
		}
		for _, c := range ds.LODs {
			if err := checkID(c.ID); err != nil {
				return err
			}
			total += len(c.Pixels)
			if c.Hold != 0 {
				return fmt.Errorf("holds belong to track frames")
			}
			if c.Anchor != nil && (c.Anchor.X < -MaxDimension || c.Anchor.X > MaxDimension || c.Anchor.Y < -MaxDimension || c.Anchor.Y > MaxDimension) {
				return fmt.Errorf("anchor outside supported logical range")
			}
			if len(c.Tracks) > 256 {
				return fmt.Errorf("too many tracks")
			}
			names := map[string]bool{}
			for _, t := range c.Tracks {
				if !ValidName(t.Name) || names[t.Name] || len(t.Frames) == 0 || len(t.Frames) > 4096 {
					return fmt.Errorf("invalid track %q", t.Name)
				}
				names[t.Name] = true
				for _, f := range t.Frames {
					frames++
					total += len(f.Pixels)
					if total > MaxCells || frames > 16384 {
						return fmt.Errorf("animation exceeds document budget")
					}
					if f.W != c.W || f.H != c.H || f.Mode != c.Mode || len(f.Tracks) > 0 || f.Anchor != nil || f.ID != "" || f.Hold > 1000000 {
						return fmt.Errorf("frame must match size/mode, inherit anchor and contain no tracks")
					}
					f.Hold = 0
					tmp := Document{Palette: d.Palette, PaletteMode: d.PaletteMode, Designs: []Design{{Name: "frame", LODs: []Canvas{f}}}}
					if err := tmp.Validate(); err != nil {
						return err
					}
				}
			}
		}
	}
	if total > MaxCells {
		return fmt.Errorf("document exceeds cell budget")
	}
	return nil
}

type wireFrame struct {
	Hold  *uint32    `json:"hold,omitempty"`
	Rows  [][]uint32 `json:"rows"`
	Tiles []Tile     `json:"tiles,omitempty"`
}
type wireTrack struct {
	Name   string      `json:"name"`
	Frames []wireFrame `json:"frames"`
}
type wireSize struct {
	wireCanvas
	ID     string      `json:"id,omitempty"`
	Anchor *Point      `json:"anchor,omitempty"`
	Tracks []wireTrack `json:"tracks,omitempty"`
}
type wireDesign struct {
	ID   string     `json:"id,omitempty"`
	Name string     `json:"name"`
	LODs []wireSize `json:"lods"`
}
type wireDocument struct {
	Format      string                     `json:"format"`
	Version     int                        `json:"version"`
	Palette     []string                   `json:"palette"`
	PaletteMode string                     `json:"palette_mode"`
	Slots       map[string]uint32          `json:"slots,omitempty"`
	Extensions  map[string]json.RawMessage `json:"extensions,omitempty"`
	Designs     []wireDesign               `json:"designs"`
}

func runs(c Canvas) [][]uint32 {
	rows := make([][]uint32, c.H)
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; {
			v := c.Pixels[y*c.W+x]
			n := 1
			for x+n < c.W && c.Pixels[y*c.W+x+n] == v {
				n++
			}
			rows[y] = append(rows[y], uint32(n), v)
			x += n
		}
	}
	return rows
}
func encodeDocument(w io.Writer, d *Document) error {
	out := wireDocument{Format: "dotmason", Version: Version, Palette: d.Palette, PaletteMode: d.PaletteMode, Slots: d.Slots, Extensions: d.Extensions}
	for _, ds := range d.Designs {
		wd := wireDesign{ID: ds.ID, Name: ds.Name}
		for _, c := range ds.LODs {
			wc := wireSize{wireCanvas: wireCanvas{Name: c.Name, Mode: c.Mode, W: c.W, H: c.H, Tiles: c.Tiles, Rows: runs(c)}, ID: c.ID, Anchor: c.Anchor}
			for _, t := range c.Tracks {
				wt := wireTrack{Name: t.Name}
				for _, f := range t.Frames {
					wt.Frames = append(wt.Frames, wireFrame{Hold: holdPointer(f.Hold), Rows: runs(f), Tiles: f.Tiles})
				}
				wc.Tracks = append(wc.Tracks, wt)
			}
			wd.LODs = append(wd.LODs, wc)
		}
		out.Designs = append(out.Designs, wd)
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(out)
}
func decodeDocument(b []byte) (*Document, error) {
	var w wireDocument
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if e := dec.Decode(&w); e != nil {
		return nil, e
	}
	if dec.Decode(new(any)) != io.EOF {
		return nil, fmt.Errorf("trailing data")
	}
	if w.Format != "dotmason" || w.Version != Version {
		return nil, fmt.Errorf("unsupported Dotmason format/version: %q v%d (expected dotmason v1)", w.Format, w.Version)
	}
	d := &Document{Palette: w.Palette, PaletteMode: w.PaletteMode, Slots: w.Slots, Extensions: w.Extensions}
	budget := 0
	expand := func(wc wireCanvas) (Canvas, error) {
		if wc.W < 1 || wc.H < 1 || wc.W > MaxDimension || wc.H > MaxDimension {
			return Canvas{}, fmt.Errorf("invalid dimensions")
		}
		budget += wc.W * wc.H
		if budget > MaxCells {
			return Canvas{}, fmt.Errorf("document exceeds cell budget")
		}
		c := Canvas{Name: wc.Name, Mode: wc.Mode, W: wc.W, H: wc.H, Tiles: wc.Tiles}
		if len(wc.Rows) != wc.H {
			return c, fmt.Errorf("row count mismatch")
		}
		c.Pixels = make([]uint32, 0, wc.W*wc.H)
		for _, row := range wc.Rows {
			if len(row)%2 != 0 {
				return c, fmt.Errorf("odd RLE row")
			}
			n := 0
			for i := 0; i < len(row); i += 2 {
				v := row[i]
				if v == 0 || v > uint32(wc.W-n) {
					return c, fmt.Errorf("invalid RLE run")
				}
				n += int(v)
				for j := uint32(0); j < v; j++ {
					c.Pixels = append(c.Pixels, row[i+1])
				}
			}
			if n != wc.W {
				return c, fmt.Errorf("row width mismatch")
			}
		}
		return c, nil
	}
	for _, ds := range w.Designs {
		nd := Design{ID: ds.ID, Name: ds.Name}
		for _, wc := range ds.LODs {
			c, e := expand(wc.wireCanvas)
			if e != nil {
				return nil, e
			}
			c.ID = wc.ID
			c.Anchor = wc.Anchor
			for _, wt := range wc.Tracks {
				t := Track{Name: wt.Name}
				for _, wf := range wt.Frames {
					f, e := expand(wireCanvas{Name: "Frame", Mode: c.Mode, W: c.W, H: c.H, Rows: wf.Rows, Tiles: wf.Tiles})
					if e != nil {
						return nil, e
					}
					if wf.Hold != nil {
						if *wf.Hold == 0 {
							return nil, fmt.Errorf("explicit hold must be positive")
						}
						f.Hold = *wf.Hold
					}
					t.Frames = append(t.Frames, f)
				}
				c.Tracks = append(c.Tracks, t)
			}
			nd.LODs = append(nd.LODs, c)
		}
		d.Designs = append(d.Designs, nd)
	}
	if e := d.Validate(); e != nil {
		return nil, e
	}
	return d, nil
}

func holdPointer(n uint32) *uint32 {
	if n == 0 {
		return nil
	}
	return &n
}
