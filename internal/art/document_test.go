package art

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExamplesRoundTrip(t *testing.T) {
	paths, _ := filepath.Glob("../examples/*.mason")
	for _, p := range paths {
		t.Run(filepath.Base(p), func(t *testing.T) {
			d, e := Load(p)
			if e != nil {
				t.Fatal(e)
			}
			var b bytes.Buffer
			if e = Encode(&b, d); e != nil {
				t.Fatal(e)
			}
			got, e := Decode(&b)
			if e != nil {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(d, got) {
				t.Fatal("roundtrip changed art")
			}
		})
	}
}
func TestRejectMalformed(t *testing.T) {
	good := `{"format":"dotmason","version":1,"palette":["#FFFFFF"],"palette_mode":"fixed","designs":[{"name":"X","lods":[{"name":"S","mode":"half","w":2,"h":1,"rows":[[2,1]]}]}]}`
	for _, bad := range []string{
		strings.Replace(good, `"version":1`, `"version":99`, 1),
		strings.Replace(good, `[2,1]`, `[0,1]`, 1),
		strings.Replace(good, `[2,1]`, `[3,1]`, 1),
		strings.Replace(good, `[2,1]`, `[1,1]`, 1),
		strings.Replace(good, `[2,1]`, `[2,2]`, 1),
		strings.Replace(good, `[2,1]`, `[2]`, 1),
		strings.Replace(good, `"w":2`, `"w":2049`, 1),
		strings.Replace(good, `"mode":"half"`, `"mode":"other"`, 1),
		strings.Replace(good, `"name":"X"`, `"name":"\u001b"`, 1),
		strings.Replace(good, `"format":`, `"unknown":0,"format":`, 1), good + `{}`,
	} {
		if _, e := Decode(strings.NewReader(bad)); e == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}
func TestUnicodeAndOddHalfRender(t *testing.T) {
	d := New()
	c, _ := NewCanvas("odd", "half", 1, 3)
	c.Pixels = []uint32{1, 2, 3}
	d.Designs[0].LODs = []Canvas{c}
	f := Render(d, &c, "#000000")
	if f.H != 2 || f.Cells[0].FG != d.Palette[0] || f.Cells[0].BG != d.Palette[1] || f.Cells[1].BG != "#000000" {
		t.Fatal(f)
	}
	c, _ = NewCanvas("wide", "unicode", 4, 1)
	c.Tiles = []Tile{{Glyph: "界", FG: 1}}
	c.Pixels[1] = 1
	d.Designs[0].LODs = []Canvas{c}
	if e := d.Validate(); e != nil {
		t.Fatal(e)
	}
	f = Render(d, &c, "#000000")
	if f.Plain() != " 界 " || f.Cells[2].Span != 0 {
		t.Fatal(f)
	}
	clipped := RenderRegion(d, &c, "#000000", 2, 0, 2, 1)
	if clipped.Plain() != "  " {
		t.Fatal("split wide glyph", clipped.Plain())
	}
	c.Pixels[3] = 1
	d.Designs[0].LODs[0] = c
	if d.Validate() == nil {
		t.Fatal("accepted dangling wide glyph")
	}
	for _, s := range []string{"\x1b", "a\nb", "ab", "\u200b", ""} {
		if _, e := GlyphWidth(s); e == nil {
			t.Fatal("accepted unsafe glyph", s)
		}
	}
	for _, s := range []string{"é", "e\u0301", "界", "👩‍🚀"} {
		if _, e := GlyphWidth(s); e != nil {
			t.Fatal(s, e)
		}
	}
}
func TestFailedSavePreservesFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "art.mason")
	d := New()
	if e := Save(p, d); e != nil {
		t.Fatal(e)
	}
	before, _ := os.ReadFile(p)
	d.Designs[0].LODs[0].Pixels[0] = 999
	if Save(p, d) == nil {
		t.Fatal("saved invalid art")
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("clobbered file")
	}
}
func TestFrameWideOverwrite(t *testing.T) {
	f := NewFrame(5, 1, "#000000")
	f.Put(1, 0, "界", "#FFFFFF", "#000000")
	f.Put(2, 0, "X", "#FFFFFF", "#000000")
	if f.Plain() != "  X  " {
		t.Fatal(f.Plain())
	}
	f.Put(1, 0, "界", "#FFFFFF", "#000000")
	f.Put(0, 0, "界", "#FFFFFF", "#000000")
	if f.Plain() != "界   " {
		t.Fatal(f.Plain())
	}
}
func FuzzDecode(f *testing.F) {
	b, _ := os.ReadFile("../examples/unicode.mason")
	f.Add(b)
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		d, e := Decode(bytes.NewReader(b))
		if e != nil {
			return
		}
		var out bytes.Buffer
		if e = Encode(&out, d); e != nil {
			t.Fatal(e)
		}
		if _, e = Decode(&out); e != nil {
			t.Fatal(e)
		}
	})
}
func BenchmarkDecodeLogos(b *testing.B) {
	data, e := os.ReadFile("../examples/ascii-bowl.mason")
	if e != nil {
		b.Fatal(e)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, e := Decode(bytes.NewReader(data)); e != nil {
			b.Fatal(e)
		}
	}
}
func BenchmarkRenderLarge(b *testing.B) {
	d, e := Load("../examples/ascii-bowl.mason")
	if e != nil {
		b.Fatal(e)
	}
	c := &d.Designs[0].LODs[1]
	b.ReportAllocs()
	for b.Loop() {
		Render(d, c, "#000000").ANSI()
	}
}
