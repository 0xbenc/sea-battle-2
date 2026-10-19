package visual

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestMenuIdlePresentation(t *testing.T) {
	for _, sd := range []bool{true, false} {
		for _, at := range []float64{0, 6, 35, 67, 180, 3600} {
			f := RenderMenu(sd, at, "")
			for _, label := range []string{"OPERATION: UNEXPECTED GUESTS", "S  Single player", "H  Host a battle", "J  Join by invitation", "Q quit"} {
				if !strings.Contains(f.Plain(), label) {
					t.Fatalf("SD=%v missing %s", sd, label)
				}
			}
			if strings.Contains(f.Plain(), "F2") {
				t.Fatal("removed toggle advertised")
			}
			RenderMenu(!sd, at+13, "")
			if !reflect.DeepEqual(f, RenderMenu(sd, at, "")) {
				t.Fatal("render depends on prior frames or size")
			}
		}
	}
}
func TestDirectorRandomPacing(t *testing.T) {
	seen := map[idleKind]bool{}
	starts := map[float64]bool{}
	for seed := uint64(1); seed <= 24; seed++ {
		d := NewMenuDirector(seed)
		starts[d.story.start] = true
		if d.story.start < 5 || d.story.start > 10 {
			t.Fatal("opening wait outside 5–10 seconds")
		}
		for i := 0; i < 60; i++ {
			previous := d.story
			seen[previous.kind] = true
			if d.ambient.start < d.story.end() && d.ambient.end() > d.story.start {
				t.Fatal("ambient event overlaps story")
			}
			d.Advance(previous.end() - d.now + 1e-7)
			if idleFamily(d.story.kind) == idleFamily(previous.kind) {
				t.Fatal("immediate story repeat")
			}
			gap := d.story.start - previous.end()
			if gap < 10 || gap > 20 {
				t.Fatalf("quiet interval %v", gap)
			}
		}
	}
	if len(starts) != 24 {
		t.Fatal("different sessions reused opening times")
	}
	if len(seen) != 9 {
		t.Fatalf("only %d story variants scheduled", len(seen))
	}
}
func TestDirectorShowsEveryStoryBeforeRepeating(t *testing.T) {
	orders := map[[9]idleKind]bool{}
	for seed := uint64(1); seed <= 100; seed++ {
		d := NewMenuDirector(seed)
		for round := 0; round < 10; round++ {
			seen := map[idleKind]bool{}
			var order [9]idleKind
			for i := range order {
				k := d.story.kind
				if seen[k] {
					t.Fatalf("seed %d round %d repeated story %d before exhausting pool", seed, round, k)
				}
				seen[k], order[i] = true, k
				previous := k
				d.Advance(d.story.end() - d.now + 1e-7)
				if idleFamily(previous) == idleFamily(d.story.kind) {
					t.Fatal("consecutive story family, including round boundary")
				}
			}
			orders[order] = true
		}
	}
	if len(orders) < 100 {
		t.Fatal("rounds reuse a fixed order")
	}
}

func TestDirectorFrameRateAndRenderingIndependent(t *testing.T) {
	a, b := NewMenuDirector(92), NewMenuDirector(92)
	a.Advance(7200)
	for i := 0; i < 72000; i++ {
		b.Advance(.1)
	}
	if a.story != b.story || a.ambient != b.ambient || a.sheen != b.sheen {
		t.Fatal("schedule depends on update rate")
	}
	before := a.Snapshot()
	RenderMenu(true, 7200, "", before)
	RenderMenu(false, 7200, "", before)
	if a.Snapshot() != before {
		t.Fatal("render consumed randomness")
	}
	a.Advance(-1)
	a.Advance(math.NaN())
	a.Advance(math.Inf(1))
	if a.Snapshot() != before {
		t.Fatal("invalid delta changed schedule")
	}
}
func TestOnlySurvivingSortiesReturn(t *testing.T) {
	for _, win := range []bool{false, true} {
		for _, age := range []float64{43 + sortieLiftTime, 48 + sortieLiftTime} {
			r := New(80, 24)
			r.idleSortie(sortieState{age: age, win: win})
			visible := false
			for _, p := range r.Pixels {
				visible = visible || p != 0
			}
			if visible != win {
				t.Fatalf("age %v: returned=%v survived=%v", age, visible, win)
			}
		}
	}
}
func TestHighDogfightOccludedByTitle(t *testing.T) {
	for _, sd := range []bool{true, false} {
		w, h, row := 160, 48, 5
		if sd {
			w, h, row = 80, 24, 3
		}
		covered := false
		for _, high := range []bool{false, true} {
			for age := 24.0; age < 28; age += .25 {
				sortie := New(w, h)
				sortie.idleSortie(sortieState{age: age + sortieLiftTime, high: high, win: true})
				title := New(w, h)
				title.menuTitle(row, -1)
				composite := New(w, h)
				copy(composite.Pixels, sortie.Pixels)
				composite.menuTitle(row, -1)
				for i, c := range title.Pixels {
					if c == 0 {
						continue
					}
					if composite.Pixels[i] != c {
						t.Fatal("aircraft painted over title")
					}
					if sortie.Pixels[i] != 0 {
						if !high {
							t.Fatal("low route crossed title")
						}
						covered = true
					}
				}
			}
		}
		if !covered {
			t.Fatalf("SD=%v high flight never crossed letters", sd)
		}
	}
}
func TestBroodMissileTrajectory(t *testing.T) {
	x, y := broodArc(68, 12, 19, 0)
	if x != 68 || y != 21 {
		t.Fatal("missile does not leave launch sac")
	}
	x, y = broodArc(68, 12, 19, 1)
	if x != 12 || y != 19 {
		t.Fatal("missile does not hit shore")
	}
	px, py := broodArc(68, 12, 19, .01)
	if math.Abs(px-68) >= math.Abs(py-21)*.1 {
		t.Fatal("launch is not near vertical")
	}
	_, apex := broodArc(68, 12, 19, .5)
	if apex >= 6 {
		t.Fatal("trajectory does not crest above title")
	}
}

func TestBombingPassClearsSubtitle(t *testing.T) {
	const subtitle = "OPERATION: UNEXPECTED GUESTS"
	for _, sd := range []bool{true, false} {
		w, h, row := 160, 48, 12
		if sd {
			w, h, row = 80, 24, 7
		}
		for age := 22.0; age < 29; age += .02 {
			r := New(w, h)
			r.idleBombingRun(age)
			left := (w - len(subtitle)) / 2
			for x := left; x < left+len(subtitle); x++ {
				for y := row * 2; y < row*2+2; y++ {
					if r.Pixels[y*w+x] != 0 {
						t.Fatalf("SD=%v bombing pass overlaps subtitle at %.2f", sd, age)
					}
				}
			}
		}
	}
}
func TestBombingSortieBranch(t *testing.T) {
	for _, sd := range []bool{true, false} {
		w, h := 160, 48
		if sd {
			w, h = 80, 24
		}
		for _, age := range []float64{24, 30, 34, 37} {
			actual, expected := New(w, h), New(w, h)
			actual.idleSortie(sortieState{age: age + sortieLiftTime, win: true, bombing: true})
			expected.idleBombingRun(age)
			if !reflect.DeepEqual(actual.Pixels, expected.Pixels) {
				t.Fatal("dogfight leaked into bombing run")
			}
		}
	}
}

var storyReviews = []struct {
	kind  idleKind
	name  string
	times []float64
}{
	{idleResupply, "resupply", []float64{2, 6, 9, 14, 16, 20, 22.5, 24.3}},
	{idleFlares, "flares", []float64{2, 4, 5, 6, 7.2, 8.3, 11, 14}},
	{idleLanding, "bad-landing", []float64{1, 3.2, 4, 5.2, 7.2, 8.5, 9.2, 12}},
	{idleBattery, "awakens", []float64{1, 3, 5.5, 6.3, 7, 8.7, 11, 15}},
	{idleSub, "sub", []float64{1, 3, 4, 6, 9.3, 11.5, 14, 18}},
	{idleBombing, "bombing", []float64{0, 2, 14, 31, 34, 36, 48, 54}},
	{idleDogfight, "dogfight", []float64{0, 2, 28, 30, 34.5, 36.3, 49, 54}},
	{idleCIWS, "ciws", []float64{1.8, 3.3, 4.7, 5.9, 6.45, 7.1, 7.8, 11.7}},
	{idleLeviathan, "leviathan", []float64{0, 1, 3, 5, 7, 9, 11, 12}},
}

func TestIdleStoriesInContext(t *testing.T) {
	for _, review := range storyReviews {
		for _, sd := range []bool{true, false} {
			for _, age := range review.times {
				state := MenuIdle{story: idleCue{review.kind, age, 256}, sheen: -1}
				f := RenderMenu(sd, age, "", state)
				if !strings.Contains(f.Plain(), "Single player") {
					t.Fatal("event obscures controls")
				}
				capture(t, f, fmt.Sprintf("story-%s-%04.1f", review.name, age), sd, 0)
			}
		}
	}
}
func TestCaptureMenuMotion(t *testing.T) {
	if os.Getenv("SB2_MENU_MOTION") == "" {
		t.Skip("opt-in animation capture")
	}
	for _, review := range storyReviews {
		if review.kind != idleBattery && review.kind != idleCIWS {
			continue
		}
		for i := 0; i < int(storyLengths[review.kind]*8); i++ {
			age := float64(i) / 8
			state := MenuIdle{story: idleCue{review.kind, age, 256}, sheen: -1}
			capture(t, RenderMenu(true, age, "", state), fmt.Sprintf("motion-%s-%04d", review.name, i), true, 0)
		}
	}
}
func BenchmarkMenuIdle(b *testing.B) {
	d := NewMenuDirector(42)
	for i := 0; i < b.N; i++ {
		d.Advance(1.0 / 24)
		RenderMenu(false, float64(i)/24, "", d.Snapshot())
	}
}

func TestFlightPathContinuity(t *testing.T) {
	for age := 0.0; age < ciwsKill; age += .02 {
		p := flightAt(ciwsDronePath, age, 0)
		if math.Hypot(p.dx, p.dy) < 6 {
			t.Fatalf("CIWS drone stalls at %.2f", age)
		}
	}
	for _, at := range []float64{2} {
		before := flightAt(ciwsDronePath, at-.001, 0)
		after := flightAt(ciwsDronePath, at+.001, 0)
		if math.Hypot(before.x-after.x, before.y-after.y) > .04 || math.Hypot(before.dx-after.dx, before.dy-after.dy) > .1 {
			t.Fatalf("CIWS drone jumps or abruptly changes velocity at %v", at)
		}
	}
	for _, high := range []bool{false, true} {
		path := func(a float64) flightPoint { return dogfightPath(a, high) }
		for _, join := range []float64{29, 32, 34} {
			before, after := flightAt(path, join-.001, 0), flightAt(path, join+.001, 0)
			if math.Hypot(after.x-before.x, after.y-before.y) > .08 {
				t.Fatal("dogfight position jumps")
			}
			if math.Hypot(after.dx-before.dx, after.dy-before.dy) > .15 {
				t.Fatal("dogfight velocity jumps")
			}
		}
	}
	before, after := flightAt(landingPath, landingTouchdown-.001, 0), flightAt(landingPath, landingTouchdown+.001, 0)
	if math.Hypot(after.dx-before.dx, after.dy-before.dy) > .1 {
		t.Fatal("landing changes speed abruptly at touchdown")
	}
	previous := landingPath(39)
	for age := 39.01; age < 47; age += .01 {
		p := landingPath(age)
		if p.x > previous.x+.001 || p.y < previous.y-.001 {
			t.Fatal("landing reverses direction")
		}
		previous = p
	}
	parked := landingPath(47)
	if math.Abs(parked.x-30) > .001 || math.Abs(parked.y-27) > .001 {
		t.Fatal("arrested aircraft misses elevator")
	}
}
func TestJetHiddenBetweenSorties(t *testing.T) {
	for _, sd := range []bool{true, false} {
		w, h := 160, 48
		if sd {
			w, h = 80, 24
		}
		for _, age := range []float64{-1, 0, 56, 57.9} {
			r := New(w, h)
			r.idleSortie(sortieState{age: age, win: true})
			for _, p := range r.Pixels {
				if p != 0 {
					t.Fatalf("jet or hatch remains visible at age %v", age)
				}
			}
		}
		r := New(w, h)
		r.menuForeground(MenuIdle{sheen: -1})
		for _, p := range r.Pixels {
			if p != 0 {
				t.Fatal("quiet menu still parks a jet on deck")
			}
		}
	}
}
func TestCarrierFlightReview(t *testing.T) {
	if os.Getenv("SB2_CAPTURE_DIR") == "" {
		t.Skip("opt-in sequence captures")
	}
	reviews := []struct {
		name  string
		times []float64
	}{
		{"turn", []float64{32.5, 33, 33.5, 34, 34.5, 35, 36, 36.5}},
		{"landing", []float64{44, 46, 47.5, 48.3, 48.5, 49, 50, 51}},
		{"elevator", []float64{.5, 1.5, 2.5, 3.5, 51.5, 52.5, 53.5, 54.5}},
	}
	for _, review := range reviews {
		for _, sd := range []bool{true, false} {
			for _, age := range review.times {
				state := MenuIdle{story: idleCue{idleDogfight, age, 256}, sheen: -1}
				capture(t, RenderMenu(sd, age, "", state), fmt.Sprintf("flight-%s-%04.1f", review.name, age), sd, 0)
			}
		}
	}
}
