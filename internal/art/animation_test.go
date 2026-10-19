package art

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func animatedFixture() *Document {
	c, _ := NewCanvas("Small", "half", 3, 2)
	c.Pixels = []uint32{1, 2, 0, 0, 2, 1}
	c.Anchor = &Point{1, 1}
	f := cloneCanvas(c)
	f.Anchor = nil
	f.Hold = 2
	g := cloneCanvas(f)
	g.Hold = 1
	g.Pixels = []uint32{0, 0, 0, 2, 0, 0}
	c.Tracks = []Track{{Name: "hit", Frames: []Canvas{f, g}}}
	large, _ := NewCanvas("Large", "half", 5, 3)
	h := cloneCanvas(large)
	h.Name = "Frame"
	large.Tracks = []Track{{Name: "hit", Frames: []Canvas{h}}, {Name: "idle", Frames: []Canvas{h}}}
	return &Document{Palette: []string{"#FFFFFF", "#FFFFFF"}, Slots: map[string]uint32{"fabric": 2}, PaletteMode: "fixed", Extensions: map[string]json.RawMessage{"example.note": json.RawMessage(`{"keep":true}`)}, Designs: []Design{{ID: "hero", Name: "Hero", LODs: []Canvas{c, large}}}}
}
func TestAnimationRoundtripAndIdentity(t *testing.T) {
	d := animatedFixture()
	var b bytes.Buffer
	if e := Encode(&b, d); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(b.String(), `"version":1`) {
		t.Fatal("not v1")
	}
	v, e := Decode(&b)
	if e != nil {
		t.Fatal(e)
	}
	// Frame names are internal; the wire format does not pretend they are identifiers.
	if len(v.Designs[0].LODs[0].Tracks[0].Frames) != 2 || len(v.Designs[0].LODs[1].Tracks[0].Frames) != 1 {
		t.Fatal("frame counts collapsed")
	}
	if !bytes.Equal(v.Extensions["example.note"], d.Extensions["example.note"]) {
		t.Fatal("extension lost")
	}
	p, e := v.ResolvePalette(map[string]string{"fabric": "#123456"})
	if e != nil || p[0] != "#FFFFFF" || p[1] != "#123456" || v.Palette[1] != "#FFFFFF" {
		t.Fatal("identity lost", p, e)
	}
	if _, e = v.ResolvePalette(map[string]string{"typo": "#123456"}); e == nil {
		t.Fatal("unknown slot accepted")
	}
	clone := v.Clone()
	clone.Designs[0].LODs[0].Tracks[0].Frames[0].Pixels[0] = 0
	clone.Slots["fabric"] = 1
	if v.Designs[0].LODs[0].Tracks[0].Frames[0].Pixels[0] != 1 || v.Slots["fabric"] != 2 {
		t.Fatal("clone aliases")
	}
}
func TestTrackClockFreeAndCoverage(t *testing.T) {
	c := animatedFixture().Designs[0].LODs[0]
	tr := c.Tracks[0]
	for unit, want := range []int{0, 0, 1, -1} {
		f, ok := tr.AtUnit(uint64(unit))
		if want < 0 {
			if ok {
				t.Fatal("implicit loop")
			}
			continue
		}
		if !ok || !reflect.DeepEqual(f, &tr.Frames[want]) {
			t.Fatal(unit)
		}
	}
	var samples []Sample
	c.Samples(func(s Sample) { samples = append(samples, s) })
	if len(samples) != 4 {
		t.Fatal("transparent pixels emitted", samples)
	}
	u, _ := NewCanvas("U", "unicode", 3, 1)
	u.Tiles = []Tile{{Glyph: "界", FG: 1}, {Glyph: " ", BG: 2}}
	u.Pixels = []uint32{1, 0, 2}
	samples = nil
	u.Samples(func(s Sample) { samples = append(samples, s) })
	if len(samples) != 2 || samples[0].Span != 2 || samples[1].BG != 2 {
		t.Fatal(samples)
	}
}
func TestAnimationRejectsLossyOrMalformed(t *testing.T) {
	cases := []func(*Document){func(d *Document) { d.Slots["bad"] = 0 }, func(d *Document) { d.Slots["alias"] = 2 }, func(d *Document) { d.Designs[0].LODs[0].Tracks[0].Frames[0].W = 4 }, func(d *Document) { d.Designs[0].LODs[0].Tracks[0].Frames = nil }, func(d *Document) { d.Designs[0].LODs[0].Anchor.X = MaxDimension + 1 }, func(d *Document) { d.Designs[0].LODs[0].Tracks[0].Frames[0].Hold = 1000001 }}
	for i, mutate := range cases {
		d := animatedFixture()
		mutate(d)
		if e := d.Validate(); e == nil {
			t.Fatal("accepted", i)
		}
	}
	var b bytes.Buffer
	Encode(&b, animatedFixture())
	s := strings.Replace(b.String(), `"version":1`, `"version":2`, 1)
	if _, e := Decode(strings.NewReader(s)); e == nil {
		t.Fatal("future version accepted")
	}
	s = strings.Replace(b.String(), `"format":"dotmason"`, `"required_future":true,"format":"dotmason"`, 1)
	if _, e := Decode(strings.NewReader(s)); e == nil {
		t.Fatal("unknown semantics accepted")
	}
}

func TestV1StrictKeysAndPositiveHold(t *testing.T) {
	var b bytes.Buffer
	Encode(&b, animatedFixture())
	for _, s := range []string{strings.Replace(b.String(), `"hold":2`, `"hold":0`, 1), strings.Replace(b.String(), `"hold":2`, `"hold":1,"hold":2`, 1)} {
		if _, e := Decode(strings.NewReader(s)); e == nil {
			t.Fatal("ambiguous file accepted")
		}
	}
}
func TestUnicodeAnimationRoundtrip(t *testing.T) {
	c, _ := NewCanvas("U", "unicode", 4, 1)
	f := c
	f.Tiles = []Tile{{Glyph: "界", FG: 2}, {Glyph: " ", BG: 1}}
	f.Pixels = []uint32{1, 0, 0, 2}
	g := c
	g.Tiles = []Tile{{Glyph: "!", FG: 1}}
	g.Pixels = []uint32{0, 1, 0, 0}
	c.Tracks = []Track{{Name: "talk", Frames: []Canvas{f, g}}}
	d := &Document{Palette: []string{"#FFFFFF", "#FFFFFF"}, PaletteMode: "fixed", Slots: map[string]uint32{"text": 2}, Designs: []Design{{Name: "Unicode", LODs: []Canvas{c}}}}
	var b bytes.Buffer
	if e := Encode(&b, d); e != nil {
		t.Fatal(e)
	}
	v, e := Decode(&b)
	if e != nil {
		t.Fatal(e)
	}
	frames := v.Designs[0].LODs[0].Tracks[0].Frames
	if frames[0].Tiles[0].Glyph != "界" || frames[1].Tiles[0].Glyph != "!" {
		t.Fatal("frame-local tiles lost")
	}
	frames[0].Pixels[1] = 2
	if e := v.Validate(); e == nil {
		t.Fatal("wide continuation overwrite accepted")
	}
}
func TestFullyTransparentUnicodeContributesNothing(t *testing.T) {
	c, _ := NewCanvas("U", "unicode", 2, 1)
	c.Tiles = []Tile{{Glyph: "界"}}
	c.Pixels = []uint32{1, 0}
	n := 0
	c.Samples(func(Sample) { n++ })
	if n != 0 {
		t.Fatal("transparent glyph emitted")
	}
}
