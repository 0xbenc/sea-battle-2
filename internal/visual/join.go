package visual

func RenderJoin(sd bool, t float64, idle MenuIdle, notice, hover string, rejoin bool) *Frame {
	r := frontOcean(sd, t, idle)
	for i, c := range r.Pixels {
		r.Pixels[i] = mix(c, Ink, .86)
	}
	left, width, scale := 16, 128, 2
	if sd {
		left, width, scale = 4, 72, 1
	}
	title := "JOIN A BATTLE"
	if rejoin {
		title = "REJOIN YOUR BATTLE"
	}
	r.text(left, 2*scale, title, Gold)
	r.factionPanel(MenuControl{X: left, Y: 8 * scale, W: width, H: 3 * scale}, Panel, .85)
	r.text(left+2, 9*scale, "Please paste an invitation.", Bone)
	r.text(left+2, 13*scale, "We'll connect automatically.", Muted)
	if rejoin {
		r.text(left+2, 15*scale, "Use a fresh invitation from the same host.", Muted)
	}
	if notice != "" && notice != "Please paste an invitation." {
		r.text(left, 18*scale, short(notice, width), Cyan)
	}
	r.pageFooter(sd, hover)
	return r.frame()
}
