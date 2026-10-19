package visual

import "fmt"

// RenderResize fits the available terminal, including very narrow windows.
func RenderResize(width, height int) *Frame {
	w, h := max(1, min(width, 160)), max(1, min(height, 24))
	r := New(w, h)
	r.rect(0, 0, w, h*2, Ink)
	center := func(row int, s string, c RGB) { r.center(row, short(s, w), c) }
	if w < 40 || h < 18 {
		lines := []string{"MORE ROOM", "Need 80 x 24", fmt.Sprintf("Now %d x %d", width, height), "Resize or zoom out", "Ctrl+C quits"}
		for i, s := range lines {
			if i < h {
				center(i, s, Bone)
			}
		}
		return r.frame()
	}
	left, boxW := max(2, (w-56)/2), min(56, w-4)
	top := max(0, (h-17)/2)
	center(top, "A LITTLE MORE ROOM", Gold)
	center(top+2, "Enlarge the window or zoom out.", Bone)
	r.factionPanel(MenuControl{X: left, Y: top + 4, W: boxW, H: 7}, Panel, .96)
	meter := func(row int, label string, current, need int) {
		color := Rust
		if current >= need {
			color = Cyan
		}
		r.text(left+2, row, label, Muted)
		value := fmt.Sprintf("%d / %d", current, need)
		r.text(left+boxW-2-len(value), row, value, color)
		barW := boxW - 4
		r.rect(left+2, (row+1)*2, barW, 1, mix(Muted, Ink, .75))
		r.rect(left+2, (row+1)*2, min(barW, max(0, current)*barW/need), 1, color)
	}
	meter(top+5, "WIDTH", width, 80)
	meter(top+8, "HEIGHT", height, 24)
	center(top+12, "Need 80 x 24 to play.", Muted)
	center(top+13, "The game returns when it fits.", Muted)
	r.line(left, (top+15)*2, left+boxW-1, (top+15)*2, mix(Muted, Ink, .65))
	r.text(left, top+16, "Ctrl+C Quit", Bone)
	r.text(left+boxW-12, top+16, "by TTY.games", Muted)
	return r.frame()
}
