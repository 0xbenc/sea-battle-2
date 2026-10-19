package visual

import (
	"strings"
	"testing"
)

func TestFactionSelectionPresentation(t *testing.T) {
	for _, sd := range []bool{true, false} {
		for _, state := range []struct{ name, hover, pressed string }{{"idle", "", ""}, {"human-hover", "a", ""}, {"brood-press", "b", "b"}} {
			f := RenderFactionMenu(sd, 0, "medium", state.hover, state.pressed, MenuIdle{sheen: -1})
			for _, text := range []string{"A  UNITED STATES NAVY", "B  THE BROOD", "DIFFICULTY", "ESC Back", "Q Quit"} {
				if !strings.Contains(f.Plain(), text) {
					t.Fatal("missing selector text", sd, text)
				}
			}
			capture(t, f, "faction-"+state.name, sd, 0)
		}
		for _, c := range FactionControls(sd) {
			for y := c.Y; y < c.Y+c.H; y++ {
				for x := c.X; x < c.X+c.W; x++ {
					if FactionControlAt(sd, x, y) != c.Key {
						t.Fatal("overlapping selector targets")
					}
				}
			}
		}
	}
}
