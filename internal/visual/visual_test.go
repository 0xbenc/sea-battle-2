package visual

import (
	"fmt"
	"git.occ.farm/occ/dotmason/art"
	"github.com/rivo/uniseg"
	"reflect"
	"sea-battle-2/internal/assets"
	"strings"
	"testing"
)

func TestSceneMatrix(t *testing.T) {
	ids := map[string]bool{}
	for i, s := range Scenes {
		if ids[s.ID] {
			t.Fatalf("duplicate scene %s", s.ID)
		}
		ids[s.ID] = true
		for _, sd := range []bool{false, true} {
			for _, at := range []float64{0, 1.6, 3, 7, s.Duration} {
				st := DefaultState()
				st.Scene = i
				st.SD = sd
				st.T = at
				f := Render(st)
				wantW, wantH := 160, 48
				if sd {
					wantW, wantH = 80, 24
				}
				if f.W != wantW || f.H != wantH || len(f.Cells) != wantW*wantH {
					t.Fatal("invalid canvas")
				}
				for y, line := range strings.Split(f.Plain(), "\n") {
					if uniseg.StringWidth(line) != wantW {
						t.Fatalf("%s SD=%v at %.1f row %d width %d", s.ID, sd, at, y, uniseg.StringWidth(line))
					}
				}
				for _, c := range f.Cells {
					if c.FG > 0xffffff || c.BG > 0xffffff {
						t.Fatal("invalid color")
					}
				}
				if !strings.Contains(f.Plain(), "SEA BATTLE 2") || !strings.Contains(f.Plain(), "SCENES") {
					t.Fatal("missing navigation")
				}
			}
		}
	}
}
func TestCameraAndWeatherMatrix(t *testing.T) {
	for _, sd := range []bool{false, true} {
		for _, camera := range []string{"TOP", "SIDE", "FRONT"} {
			for weather := 0; weather < 4; weather++ {
				for _, id := range []string{"ford", "hive", "hit", "sinking"} {
					s := DefaultState()
					s.Scene = Find(id)
					s.SD = sd
					s.Camera = camera
					s.Weather = weather
					s.T = 2.3
					Render(s)
				}
			}
		}
	}
}
func TestDeterminismAndMotion(t *testing.T) {
	s := DefaultState()
	s.Scene = Find("exchange")
	s.T = 5
	if !reflect.DeepEqual(Render(s), Render(s)) {
		t.Fatal("render is not deterministic")
	}
	before := Render(s)
	s.T += .2
	if reflect.DeepEqual(before, Render(s)) {
		t.Fatal("pursuit does not animate")
	}
	s.Scene = Find("sinking")
	s.T = 3
	before = Render(s)
	s.T = 11
	if reflect.DeepEqual(before, Render(s)) {
		t.Fatal("sinking does not advance")
	}
}
func TestHalfPixelCoveragePreservesOcean(t *testing.T) {
	r := New(4, 2)
	r.rect(0, 0, 4, 4, 0x123456)
	c := &art.Canvas{Mode: "half", W: 2, H: 2, Pixels: []uint32{0, 1, 1, 0}}
	r.asset(c, 1, 1, 0, 0, false)
	if r.Pixels[1*r.W+1] != 0x123456 || r.Pixels[2*r.W+2] != 0x123456 {
		t.Fatal("transparent art erased ocean")
	}
	if r.Pixels[1*r.W+2] != RGB(assets.Palette[1]) {
		t.Fatal("opaque coverage lost")
	}
	r.asset(c, -1, -1, 0, 0, false) // clipped partial sprite must be safe
}
func TestHostileBoardDoesNotRevealFleet(t *testing.T) {
	st := DefaultState()
	st.SD = true
	a, b := New(80, 24), New(80, 24)
	a.board(st, 5, 8, 3, 2, false, 0)
	b.board(st, 5, 8, 3, 2, false, 1)
	if !reflect.DeepEqual(a.frame(), b.frame()) {
		t.Fatal("hostile board depends on hidden fleet identity")
	}
	a, b = New(80, 24), New(80, 24)
	a.board(st, 5, 8, 3, 2, true, 0)
	b.board(st, 5, 8, 3, 2, true, 1)
	if reflect.DeepEqual(a.frame(), b.frame()) {
		t.Fatal("friendly fleets must differ")
	}
}
func TestFleetArtwork(t *testing.T) {
	for i := 0; i < 10; i++ {
		for _, view := range []string{"top", "side", "front"} {
			for _, size := range []string{"HD", "SD"} {
				c := assets.Get(fmt.Sprintf("ship-%d-%s", i, view), size)
				paint := 0
				for _, p := range c.Pixels {
					if p != 0 {
						paint++
					}
				}
				if paint < c.W*c.H/12 {
					t.Fatalf("empty silhouette: %d %s %s", i, view, size)
				}
			}
		}
	}
}

func TestPlacementGhostHonorsOccupiedSquares(t *testing.T) {
	for _, tc := range []struct {
		x, y       int
		rot, valid bool
	}{{6, 4, false, true}, {9, 7, false, false}, {6, 4, true, false}, {0, 1, false, false}, {6, 8, true, true}} {
		if placementLegal(tc.x, tc.y, tc.rot) != tc.valid {
			t.Fatalf("unexpected placement: %+v", tc)
		}
	}
}

func TestSequenceOutcomeIsVisibleOnFinalPlot(t *testing.T) {
	for _, miss := range []bool{false, true} {
		s := DefaultState()
		s.SD = true
		s.Scene = Find("exchange")
		s.T = 14
		s.ShotMiss = miss
		f := Render(s)
		want := "HIT CONFIRMED"
		if miss {
			want = "MISS CONFIRMED"
		}
		if !strings.Contains(f.Plain(), want) {
			t.Fatal("sequence lost outcome", want)
		}
	}
}
func BenchmarkHDCinematic(b *testing.B) {
	s := DefaultState()
	s.Scene = Find("hit")
	s.T = 2.1
	for b.Loop() {
		Render(s).ANSI()
	}
}

func TestEffectColorSlotCannotRecolorShips(t *testing.T) {
	foam := assets.Doc.Slots["foam"]
	if foam == 0 {
		t.Fatal("missing foam identity")
	}
	for _, d := range assets.Doc.Designs {
		if !strings.HasPrefix(d.ID, "ship-") {
			continue
		}
		for _, c := range d.LODs {
			for _, index := range c.Pixels {
				if index == foam {
					t.Fatalf("foam override would recolor %s/%s", d.ID, c.Name)
				}
			}
		}
	}
}
