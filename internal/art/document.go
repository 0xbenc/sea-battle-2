// Package art loads, validates and renders Dotmason artwork independently of
// the editor. Palette index zero is transparent; real colors are one-based.
package art

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

const (
	Version      = 1
	MaxDimension = 2048
	MaxCells     = 4 * 1024 * 1024
	MaxFileBytes = 32 * 1024 * 1024
)

var hexColor = regexp.MustCompile("^#[0-9a-fA-F]{6}$")

type Tile struct {
	Glyph string `json:"g"`
	FG    uint32 `json:"f"`
	BG    uint32 `json:"b,omitempty"`
}
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}
type Track struct {
	Name   string
	Frames []Canvas
}
type Canvas struct {
	ID     string
	Anchor *Point
	Hold   uint32 // zero means one unit; never seconds
	Tracks []Track
	Name   string
	Mode   string // half or unicode
	W, H   int
	Pixels []uint32 // palette indices in half; tile indices in unicode
	Tiles  []Tile   // one-based; zero is an empty terminal cell
}
type Design struct {
	ID   string
	Name string
	LODs []Canvas
}
type Document struct {
	Slots       map[string]uint32          // explicit overridable palette identities
	Extensions  map[string]json.RawMessage // namespaced, nonsemantic, preserved
	Palette     []string
	PaletteMode string
	Designs     []Design
}
type wireCanvas struct {
	Name  string     `json:"name"`
	Mode  string     `json:"mode"`
	W     int        `json:"w"`
	H     int        `json:"h"`
	Tiles []Tile     `json:"tiles,omitempty"`
	Rows  [][]uint32 `json:"rows"`
}

func NewCanvas(name, mode string, w, h int) (Canvas, error) {
	if w < 1 || h < 1 || w > MaxDimension || h > MaxDimension || w*h > MaxCells {
		return Canvas{}, errors.New("canvas dimensions must be 1..2048")
	}
	if mode != "half" && mode != "unicode" {
		return Canvas{}, errors.New("mode must be half or unicode")
	}
	return Canvas{Name: name, Mode: mode, W: w, H: h, Pixels: make([]uint32, w*h)}, nil
}
func New() *Document {
	c, _ := NewCanvas("Small", "half", 16, 16)
	return &Document{Palette: []string{"#F2F0E5", "#E9B44C", "#4DC9B0", "#E05A6F", "#526DCC", "#263342"}, PaletteMode: "unlimited", Designs: []Design{{Name: "Untitled", LODs: []Canvas{c}}}}
}
func ValidColor(s string) bool { return hexColor.MatchString(s) }
func ValidName(s string) bool {
	if s == "" || len(s) > 128 || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}
func GlyphWidth(s string) (int, error) {
	if s == "" || len(s) > 128 || !utf8.ValidString(s) || uniseg.GraphemeClusterCount(s) != 1 {
		return 0, errors.New("enter exactly one Unicode grapheme")
	}
	for _, r := range s {
		if unicode.IsControl(r) || (unicode.Is(unicode.Cf, r) && r != '\u200d') {
			return 0, errors.New("control characters are not paintable")
		}
	}
	w := uniseg.StringWidth(s)
	if w < 1 || w > 2 {
		return 0, errors.New("glyph must occupy one or two terminal columns")
	}
	return w, nil
}
func (c Canvas) TermHeight() int {
	if c.Mode == "half" {
		return (c.H + 1) / 2
	}
	return c.H
}
func (c Canvas) TileAt(x, y int) Tile {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return Tile{Glyph: " "}
	}
	n := c.Pixels[y*c.W+x]
	if n == 0 || int(n) > len(c.Tiles) {
		return Tile{Glyph: " "}
	}
	return c.Tiles[n-1]
}
func (c Canvas) Continuation(x, y int) bool {
	if c.Mode != "unicode" || x < 1 {
		return false
	}
	t := c.TileAt(x-1, y)
	return uniseg.StringWidth(t.Glyph) == 2
}
func (d *Document) Validate() error {
	if d == nil {
		return errors.New("nil document")
	}
	if d.PaletteMode != "fixed" && d.PaletteMode != "unlimited" {
		return errors.New("palette_mode must be fixed or unlimited")
	}
	if len(d.Palette) > MaxCells {
		return errors.New("palette too large")
	}
	for _, s := range d.Palette {
		if !ValidColor(s) {
			return fmt.Errorf("invalid RGB color %q", s)
		}
	}
	if len(d.Designs) < 1 || len(d.Designs) > 256 {
		return errors.New("document needs 1..256 designs")
	}
	total := 0
	names := map[string]bool{}
	for _, design := range d.Designs {
		if !ValidName(design.Name) || names[design.Name] {
			return errors.New("design names must be unique, nonempty and free of controls")
		}
		names[design.Name] = true
		if len(design.LODs) < 1 || len(design.LODs) > 64 {
			return errors.New("design needs 1..64 LODs")
		}
		lods := map[string]bool{}
		for _, c := range design.LODs {
			if !ValidName(c.Name) || lods[c.Name] {
				return errors.New("LOD names must be unique and nonempty")
			}
			lods[c.Name] = true
			if c.W < 1 || c.H < 1 || c.W > MaxDimension || c.H > MaxDimension || len(c.Pixels) != c.W*c.H {
				return errors.New("invalid canvas dimensions or pixel count")
			}
			total += len(c.Pixels)
			if total > MaxCells {
				return errors.New("document exceeds 4 million logical pixels/cells")
			}
			if c.Mode != "half" && c.Mode != "unicode" {
				return errors.New("invalid canvas mode")
			}
			if c.Mode == "half" && len(c.Tiles) > 0 {
				return errors.New("half canvases cannot have tiles")
			}
			if len(c.Tiles) > MaxCells {
				return errors.New("tile table too large")
			}
			for _, t := range c.Tiles {
				if _, err := GlyphWidth(t.Glyph); err != nil {
					return err
				}
				if int(t.FG) > len(d.Palette) || int(t.BG) > len(d.Palette) {
					return errors.New("tile palette index out of range")
				}
			}
			limit := len(d.Palette)
			if c.Mode == "unicode" {
				limit = len(c.Tiles)
			}
			for i, n := range c.Pixels {
				if int(n) > limit {
					return errors.New("paint index out of range")
				}
				if c.Mode == "unicode" && n > 0 {
					w, _ := GlyphWidth(c.Tiles[n-1].Glyph)
					if w == 2 && (i%c.W == c.W-1 || c.Pixels[i+1] != 0) {
						return errors.New("wide glyph requires an empty continuation cell inside the row")
					}
				}
			}
		}
	}
	return d.validateAnimation()
}
func Decode(r io.Reader) (*Document, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxFileBytes {
		return nil, errors.New("file exceeds 32 MiB")
	}
	if e := uniqueKeys(b); e != nil {
		return nil, e
	}
	return decodeDocument(b)
}
func Encode(w io.Writer, d *Document) error {
	if err := d.Validate(); err != nil {
		return err
	}
	return encodeDocument(w, d)
}
func Load(path string) (*Document, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return Decode(f)
}

// Save replaces the file only after a complete validated write and fsync.
// The editor detects external modifications separately before calling this.
func Save(path string, d *Document) error {
	if path == "" {
		return errors.New("choose a filename")
	}
	var b bytes.Buffer
	if err := Encode(&b, d); err != nil {
		return err
	}
	if b.Len() > MaxFileBytes {
		return errors.New("encoded file exceeds 32 MiB")
	}
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".dotmason-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(b.Bytes()); err == nil {
		err = f.Sync()
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}
func cloneCanvas(c Canvas) Canvas {
	c.Pixels = append([]uint32(nil), c.Pixels...)
	c.Tiles = append([]Tile(nil), c.Tiles...)
	if c.Anchor != nil {
		p := *c.Anchor
		c.Anchor = &p
	}
	c.Tracks = append([]Track(nil), c.Tracks...)
	for i := range c.Tracks {
		c.Tracks[i].Frames = append([]Canvas(nil), c.Tracks[i].Frames...)
		for j := range c.Tracks[i].Frames {
			c.Tracks[i].Frames[j] = cloneCanvas(c.Tracks[i].Frames[j])
		}
	}
	return c
}
func (d *Document) Clone() *Document {
	out := &Document{Palette: append([]string(nil), d.Palette...), PaletteMode: d.PaletteMode}
	if d.Slots != nil {
		out.Slots = map[string]uint32{}
		for k, v := range d.Slots {
			out.Slots[k] = v
		}
	}
	if d.Extensions != nil {
		out.Extensions = map[string]json.RawMessage{}
		for k, v := range d.Extensions {
			out.Extensions[k] = append(json.RawMessage(nil), v...)
		}
	}
	for _, ds := range d.Designs {
		nd := Design{ID: ds.ID, Name: ds.Name}
		for _, c := range ds.LODs {
			nd.LODs = append(nd.LODs, cloneCanvas(c))
		}
		out.Designs = append(out.Designs, nd)
	}
	return out
}
func (d *Document) Color(n uint32, backdrop string) string {
	if n == 0 || int(n) > len(d.Palette) {
		return backdrop
	}
	return strings.ToUpper(d.Palette[n-1])
}
