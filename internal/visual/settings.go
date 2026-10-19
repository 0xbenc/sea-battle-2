package visual

func SettingsControls(sd bool) []MenuControl {
	left, width, scale := 16, 128, 2
	if sd {
		left, width, scale = 4, 72, 1
	}
	controls := []MenuControl{
		{Key: "d", X: left, Y: 5 * scale, W: width, H: 2 * scale},
		{Key: "m", X: left, Y: 8 * scale, W: width, H: 2 * scale},
		{Key: "c", X: left, Y: 17 * scale, W: width, H: 2 * scale},
	}
	for i, label := range []string{"1 Easy", "2 Medium", "3 Hard"} {
		controls = append(controls, MenuControl{Key: string(rune('1' + i)), Label: label, X: left + i*(width/3), Y: 14 * scale, W: width/3 - 2, H: scale})
	}
	return append(controls, FooterControls(sd)...)
}

func SettingsControlAt(sd bool, x, y int) string {
	for _, c := range SettingsControls(sd) {
		if x >= c.X && x < c.X+c.W && y >= c.Y && y < c.Y+c.H {
			return c.Key
		}
	}
	return ""
}

func RenderSettings(sd bool, t float64, idle MenuIdle, compact, animated bool, difficulty, notice, hover string) *Frame {
	r := frontOcean(sd, t, idle)
	for i, c := range r.Pixels {
		r.Pixels[i] = mix(c, Ink, .86)
	}
	controls := SettingsControls(sd)
	left, width, scale := controls[0].X, controls[0].W, 2
	if sd {
		scale = 1
	}
	r.text(left, 2*scale, "SETTINGS", Gold)
	r.text(left, 4*scale, "DISPLAY & MOTION", Muted)
	for _, c := range controls[:6] {
		if c.Key == "d" || c.Key == "m" || c.Key == "c" {
			r.factionPanel(c, Panel, .85)
		}
		if c.Key == hover {
			r.factionPanel(c, 0xffffff, .10)
		}
	}
	toggle := func(c MenuControl, title, detail string, on bool) {
		r.text(c.X+2, c.Y, title, Bone)
		r.text(c.X+2, c.Y+scale, detail, Muted)
		x, y := c.X+c.W-10, c.Y*2
		color, label := RGB(0x42545c), "OFF"
		if on {
			color, label = RGB(0x32965d), "ON"
		}
		r.text(x-5, c.Y, label, Muted)
		r.rect(x, y, 7, 2, color)
		knob := x
		if on {
			knob = x + 5
		}
		r.rect(knob, y, 2, 2, 0xf2f1e7)
	}
	toggle(controls[0], "D  Compact display", "On: 80 x 24. Off: adapt to your terminal.", compact)
	toggle(controls[1], "M  Animated menu", "Ocean and menu motion. Battles stay animated.", animated)
	r.text(left, 12*scale, "SOLO DIFFICULTY", Gold)
	r.text(left, 13*scale, "Choose your default opponent.", Muted)
	selected := map[string]string{"easy": "1", "medium": "2", "hard": "3"}[difficulty]
	for _, c := range controls[3:6] {
		color := Muted
		r.factionPanel(c, Panel, .8)
		if c.Key == selected {
			r.factionPanel(c, Cyan, .22)
			color = Bone
		}
		if c.Key == hover {
			r.factionPanel(c, 0xffffff, .10)
		}
		prefix := "  "
		if c.Key == selected {
			prefix = "> "
		}
		r.text(c.X+2, c.Y, prefix+c.Label, color)
	}
	c := controls[2]
	r.text(c.X+2, c.Y, "C  Credits", Bone)
	r.text(c.X+2, c.Y+scale, "The people behind Sea Battle 2", Muted)
	r.text(c.X+c.W-4, c.Y, ">", Cyan)
	if notice != "" {
		r.text(left, 20*scale, short(notice, width), Cyan)
	}
	r.pageFooter(sd, hover)
	return r.frame()
}
