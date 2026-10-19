package visual

import (
	"strings"
	"testing"
)

func TestResizeFitsAvailableSpace(t *testing.T) {
	for _, size := range [][2]int{{70, 20}, {79, 24}, {80, 23}, {45, 18}, {20, 8}, {1, 1}, {0, 0}, {160, 20}} {
		f := RenderResize(size[0], size[1])
		if f.W != max(1, size[0]) || f.H != max(1, size[1]) {
			t.Fatalf("wrong dimensions: %v", size)
		}
		if size[0] >= 40 && size[1] >= 18 {
			for _, label := range []string{"WIDTH", "HEIGHT", "80 x 24", "Ctrl+C Quit"} {
				if !strings.Contains(f.Plain(), label) {
					t.Fatalf("missing %s at %v", label, size)
				}
			}
		}
	}
}
