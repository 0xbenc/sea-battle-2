package visual

import (
	"fmt"
	"testing"
)

func TestMenuReactionTargets(t *testing.T) {
	for _, sd := range []bool{true, false} {
		idle := MenuIdle{sheen: -1}
		masks := newMenuMasks(sd, idle)
		scale := masks.ship.W / 80
		if fx := MenuReactionAt(sd, idle, 5*scale, 16*scale); fx.Kind != "ripple" {
			t.Fatalf("SD=%v open water is %q", sd, fx.Kind)
		}
		for _, p := range [][2]int{{-1, 16}, {5, 0}, {0, 10}, {40, 21}} {
			if fx := MenuReactionAt(sd, idle, p[0]*scale, p[1]*scale); fx.Kind != "" {
				t.Fatalf("non-water accepted: %v %s", p, fx.Kind)
			}
		}
		for _, target := range []struct {
			kind string
			mask *Raster
		}{{"spark", masks.ship}, {"glint", masks.title}} {
			found := false
			for i, c := range target.mask.Pixels {
				if c == 0 {
					continue
				}
				x, y := i%target.mask.W, i/target.mask.W/2
				fx := MenuReactionAt(sd, idle, x, y)
				if fx.Kind != target.kind {
					continue
				}
				found = true
				fx.Age = .2
				base := RenderMenu(sd, 0, "", idle)
				effect := RenderMenuWithHover(sd, 0, "", "", idle, fx)
				changed := 0
				for n := range base.Cells {
					if base.Cells[n] != effect.Cells[n] {
						changed++
					}
				}
				if changed == 0 {
					t.Fatal("reaction invisible", target.kind)
				}
				capture(t, effect, fmt.Sprintf("click-%s", target.kind), sd, 0)
				break
			}
			if !found {
				t.Fatal("no clickable pixels", sd, target.kind)
			}
		}
		fx := MenuReactionAt(sd, idle, 5*scale, 16*scale)
		fx.Age = .2
		capture(t, RenderMenuWithHover(sd, 0, "", "", idle, fx), "click-ripple", sd, 0)
		// The surfaced submarine occupies water that is otherwise eligible.
		active := MenuIdle{story: idleCue{kind: idleSub, age: 6}, sheen: -1}
		occupied := newMenuMasks(sd, active)
		rejected := false
		for i, c := range occupied.events.Pixels {
			if c == 0 {
				continue
			}
			x, row := i%occupied.events.W, i/occupied.events.W/2
			if MenuReactionAt(sd, idle, x, row).Kind == "ripple" && MenuReactionAt(sd, active, x, row).Kind == "" {
				rejected = true
				break
			}
		}
		if !rejected {
			t.Fatal("moving object does not block water clicks")
		}
	}
}
