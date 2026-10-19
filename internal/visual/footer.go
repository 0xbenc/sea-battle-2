package visual

// FooterControls shares exact drawing and hit bounds across secondary screens.
func FooterControls(sd bool) []MenuControl {
	left, row := 16, 46
	if sd {
		left, row = 4, 22
	}
	return []MenuControl{
		{Key: "esc", Label: "ESC Back", X: left, Y: row, W: 10, H: 1},
		{Key: "q", Label: "Q Quit", X: left + 14, Y: row, W: 8, H: 1},
	}
}

func FooterControlAt(sd bool, x, y int) string {
	for _, c := range FooterControls(sd) {
		if x >= c.X && x < c.X+c.W && y >= c.Y && y < c.Y+c.H {
			return c.Key
		}
	}
	return ""
}

func (r *Raster) pageFooter(sd bool, hover string) {
	r.controlFooter(FooterControls(sd), hover)
}

func (r *Raster) controlFooter(controls []MenuControl, hover string) {
	left, row := controls[0].X, controls[0].Y
	r.line(left, (row-1)*2, r.W-left-1, (row-1)*2, mix(Muted, Ink, .65))
	for _, c := range controls {
		color := Bone
		if hover == c.Key {
			r.factionPanel(c, 0xffffff, .12)
			color = 0xffffff
		}
		r.text(c.X, c.Y, c.Label, color)
	}
	const brand = "by TTY.games"
	r.text(r.W-left-len(brand), row, brand, Muted)
}
