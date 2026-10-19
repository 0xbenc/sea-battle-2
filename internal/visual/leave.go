package visual

func LeaveControls(sd bool) []MenuControl {
	left, width, row, height := 16, 128, 28, 5
	if sd {
		left, width, row, height = 4, 72, 14, 3
	}
	return []MenuControl{
		{Key: "n", Label: "N / ESC  Stay", X: left, Y: row, W: width/2 - 2, H: height},
		{Key: "y", Label: "Y / ENTER  Leave", X: left + width/2 + 2, Y: row, W: width/2 - 2, H: height},
	}
}
func LeaveControlAt(sd bool, x, y int) string {
	for _, c := range LeaveControls(sd) {
		if x >= c.X && x < c.X+c.W && y >= c.Y && y < c.Y+c.H {
			return c.Key
		}
	}
	return ""
}
func RenderLeave(sd bool, t float64, idle MenuIdle, solo, host, menu bool, hover string) *Frame {
	r := frontOcean(sd, t, idle)
	for i, c := range r.Pixels {
		r.Pixels[i] = mix(c, Ink, .86)
	}
	controls := LeaveControls(sd)
	left, width, scale := controls[0].X, controls[1].X+controls[1].W-controls[0].X, 2
	if sd {
		scale = 1
	}
	title, detail := "LEAVE THIS BATTLE?", "You'll disconnect. Your friend can keep the session open."
	if solo {
		detail = "Your progress in this battle won't be saved."
	}
	if host && !solo {
		title = "END THIS BATTLE?"
		detail = "This ends the session for both players."
	}
	r.text(left, 2*scale, title, Gold)
	r.factionPanel(MenuControl{X: left, Y: 8 * scale, W: width, H: 3 * scale}, Panel, .95)
	r.text(left+2, 9*scale, detail, Bone)
	for _, c := range controls {
		tint := Cyan
		if c.Key == "y" {
			tint = Rust
		}
		r.factionPanel(c, tint, .12)
		color := Bone
		if hover == c.Key {
			r.factionPanel(c, 0xffffff, .12)
			color = 0xffffff
		}
		r.text(c.X+(c.W-len(c.Label))/2, c.Y+(c.H-1)/2, c.Label, color)
	}
	next := "Leave closes the game."
	if menu {
		next = "Leave returns to the main menu."
	}
	r.text(left, 19*scale, next, Muted)
	footer := FooterControls(sd)[0].Y
	r.line(left, (footer-1)*2, r.W-left-1, (footer-1)*2, mix(Muted, Ink, .65))
	r.text(left, footer, "You can stay and keep playing.", Muted)
	const brand = "by TTY.games"
	r.text(r.W-left-len(brand), footer, brand, Muted)
	return r.frame()
}
