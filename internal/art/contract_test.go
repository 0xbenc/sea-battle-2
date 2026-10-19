package art

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

// These authored fixtures predate any future writer changes. Do not regenerate
// them to accommodate breaking semantics: that would defeat the v1 lock.
func TestLockedV1Contract(t *testing.T) {
	if Version != 1 {
		t.Fatal("v1 is locked; a bump requires multi-game use and explicit review")
	}
	for _, name := range []string{"static", "animated"} {
		t.Run(name, func(t *testing.T) {
			original, err := os.ReadFile("testdata/v1/" + name + ".mason")
			if err != nil {
				t.Fatal(err)
			}
			doc, err := Decode(bytes.NewReader(original))
			if err != nil {
				t.Fatal(err)
			}
			var encoded bytes.Buffer
			if err := Encode(&encoded, doc); err != nil {
				t.Fatal(err)
			}
			var envelope struct {
				Format  string
				Version int
			}
			if err := json.Unmarshal(encoded.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Format != "dotmason" || envelope.Version != 1 {
				t.Fatal(envelope)
			}
			again, err := Decode(&encoded)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(doc, again) {
				t.Fatal("v1 roundtrip lost authored data")
			}
			for _, version := range []string{"0", "2", "3", "99"} {
				input := strings.Replace(string(original), `"version":1`, `"version":`+version, 1)
				if _, err := Decode(strings.NewReader(input)); err == nil {
					t.Fatalf("accepted unsupported version %s", version)
				}
			}
			for _, name := range []string{"cellsmith", "unknown"} {
				input := strings.Replace(string(original), `"format":"dotmason"`, `"format":"`+name+`"`, 1)
				if _, err := Decode(strings.NewReader(input)); err == nil {
					t.Fatalf("accepted foreign format %s", name)
				}
			}
		})
	}
	doc, err := Load("testdata/v1/animated.mason")
	if err != nil {
		t.Fatal(err)
	}
	small, large := doc.Designs[0].LODs[0], doc.Designs[0].LODs[1]
	if *small.Anchor != (Point{-1, 3}) || len(small.Tracks) != 1 || len(large.Tracks) != 2 || len(small.Tracks[0].Frames) != 2 || len(large.Tracks[0].Frames) != 1 {
		t.Fatal("independent size/track contract changed")
	}
	track := small.Tracks[0]
	if track.TotalUnits() != 3 {
		t.Fatal("holds changed")
	}
	frame, ok := track.AtUnit(2)
	if !ok || !reflect.DeepEqual(frame.Pixels, []uint32{0, 0, 0, 0, 2, 0}) {
		t.Fatal("unit boundary or complete-frame semantics changed")
	}
	if _, ok := track.AtUnit(3); ok {
		t.Fatal("host-owned end behavior changed")
	}
	palette, err := doc.ResolvePalette(map[string]string{"jersey": "#336699"})
	if err != nil || palette[0] != "#FFFFFF" || palette[1] != "#336699" || doc.Palette[1] != "#FFFFFF" {
		t.Fatal("fixed/variable color identity changed")
	}
	var got []Sample
	large.Tracks[0].Frames[0].Samples(func(s Sample) { got = append(got, s) })
	if len(got) != 2 || got[0].Glyph != "界" || got[0].Span != 2 || got[0].FG != 2 || got[1].X != 3 || got[1].BG != 3 {
		t.Fatal("Unicode coverage changed", got)
	}
	if string(doc.Extensions["example.note"]) != `{"preserve":[true,17,"opaque"]}` {
		t.Fatal("extension data changed")
	}
}
