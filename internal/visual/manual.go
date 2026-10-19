package visual

// RenderManual gives each phase its own control column and reserves the footer
// for navigation, even at the minimum supported terminal size.
func RenderManual(sd bool, t float64, idle MenuIdle, hover string) *Frame {
	r := frontOcean(sd, t, idle)
	for i, color := range r.Pixels {
		r.Pixels[i] = mix(color, Ink, .86)
	}
	left, right, title, rules, heading, step, network := 4, 42, 2, 4, 7, 1, 17
	if !sd {
		left, right, title, rules, heading, step, network = 16, 86, 4, 8, 14, 2, 34
	}
	r.text(left, title, "FIELD MANUAL", Gold)
	r.text(left, rules, "Sink all five enemy ships. Players alternate shots.", Bone)
	r.text(left, rules+step, "A-J = columns   1-10 = rows   B4 = column B, row 4", Muted)
	column := func(x int, label string, rows [][2]string) {
		r.text(x, heading, label, Gold)
		for i, row := range rows {
			y := heading + (i+2)*step
			r.text(x, y, row[0], Cyan)
			r.text(x+16, y, row[1], Bone)
		}
	}
	column(left, "01 / DEPLOY YOUR FLEET", [][2]string{
		{"Mouse / arrows", "Move preview"},
		{"1-5 / Tab", "Select ship"},
		{"Click / Enter", "Place ship"},
		{"Wheel / X", "Rotate ship"},
		{"R / U / Del", "Shuffle/undo/remove"},
		{"Space", "Launch fleet"},
	})
	column(right, "02 / TAKE YOUR SHOT", [][2]string{
		{"Mouse / arrows", "Aim"},
		{"Click / Enter", "Fire"},
		{"Tab", "Switch charts"},
		{"Space / Enter", "Skip cinematic"},
		{"?", "In-game help"},
		{"", "Hits end your turn."},
	})
	r.text(left, network, "03 / PLAY TOGETHER", Gold)
	r.text(left, network+step, "H hosts. Share the invitation privately. Your friend presses J.", Bone)
	r.text(left, network+2*step, "Copy or paste the invitation to connect. Keep the host running.", Muted)
	r.pageFooter(sd, hover)
	return r.frame()
}
