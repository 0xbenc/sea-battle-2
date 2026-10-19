package visual

import "strings"

// A shared two-line notice keeps feedback distinct from instructions.
func (r *Raster) noticeBox(x, row, width int, message string, color RGB) {
	r.factionPanel(MenuControl{X: x, Y: row, W: width, H: 2}, Panel, .96)
	r.rect(x, row*2, 1, 4, color)
	for i, line := range wrapMessage(message, width-4) {
		r.text(x+2, row+i, line, Bone)
	}

}

func RenderProblem(sd bool, t float64, idle MenuIdle, notice, hover string) *Frame {
	r := frontOcean(sd, t, idle)
	for i, c := range r.Pixels {
		r.Pixels[i] = mix(c, Ink, .86)
	}
	left, width, scale := 16, 128, 2
	if sd {
		left, width, scale = 4, 72, 1
	}
	r.text(left, 2*scale, "COULDN'T START THE GAME", Rust)
	r.noticeBox(left, 9*scale, width, notice, Rust)
	r.text(left+2, 14*scale, "Go back to the menu to try again.", Muted)
	r.pageFooter(sd, hover)
	return r.frame()
}

// Two lines are reserved for feedback and connection status at both sizes.
func wrapMessage(message string, width int) []string {
	words := strings.Fields(message)
	var lines []string
	for len(words) > 0 && len(lines) < 2 {
		if len(lines) == 1 {
			lines = append(lines, short(strings.Join(words, " "), width))
			break
		}
		line := words[0]
		words = words[1:]
		for len(words) > 0 && len([]rune(line+" "+words[0])) <= width {
			line += " " + words[0]
			words = words[1:]
		}
		lines = append(lines, short(line, width))
	}
	return lines
}
