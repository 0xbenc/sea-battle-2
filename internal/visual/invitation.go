package visual

import (
	"strings"
)

func InvitationControls(sd, ready bool) []MenuControl {
	left, width, scale, footer := 16, 128, 2, 46
	if sd {
		left, width, scale, footer = 4, 72, 1, 22
	}
	controls := []MenuControl{
		{Key: "enter", Label: "ENTER Arrange fleet", X: left, Y: footer, W: 21, H: 1},
		{Key: "q", Label: "Q Leave", X: left + 25, Y: footer, W: 9, H: 1},
	}
	if ready {
		controls = append(controls,
			MenuControl{Key: "c", Label: "C  Copy invitation", X: left, Y: 9 * scale, W: width/2 - 2, H: 2*scale + 1},
			MenuControl{Key: "s", Label: "S  Save invitation.txt", X: left + width/2 + 2, Y: 9 * scale, W: width/2 - 2, H: 2*scale + 1})
	}
	return controls
}
func InvitationControlAt(sd, ready bool, x, y int) string {
	for _, c := range InvitationControls(sd, ready) {
		if x >= c.X && x < c.X+c.W && y >= c.Y && y < c.Y+c.H {
			return c.Key
		}
	}
	return ""
}

func RenderInvitation(sd bool, t, progressTime float64, idle MenuIdle, ready bool, status, notice, hover string) *Frame {
	r := frontOcean(sd, t, idle)
	for i, c := range r.Pixels {
		r.Pixels[i] = mix(c, Ink, .86)
	}
	left, width, scale := 16, 128, 2
	if sd {
		left, width, scale = 4, 72, 1
	}
	title := "PREPARING YOUR INVITATION"
	if ready {
		title = "INVITE YOUR FRIEND"
	}
	r.text(left, 2*scale, title, Gold)
	statusTop := 4 * scale
	if !ready {
		statusTop = 8 * scale
	}
	r.factionPanel(MenuControl{X: left, Y: statusTop, W: width, H: 3 * scale}, Panel, .85)
	label, color := "SETTING UP YOUR CONNECTION", Cyan
	if ready {
		label, color = "YOUR INVITATION IS READY", Green
	}
	// Center the complete text block, including any wrapped connection status.
	lines := wrapMessage(status, width-14)
	blockHeight := 1
	if len(lines) > 0 {
		blockHeight = scale + len(lines)
	}
	row := statusTop + (3*scale-blockHeight)/2
	r.text(left+2, row, label, color)
	for i, line := range lines {
		r.text(left+2, row+scale+i, line, Muted)
	}
	if !ready {
		r.loadingDots(left+width-8, statusTop+(3*scale-1)/2, progressTime)
	}

	if strings.HasPrefix(notice, "Saved to: ") {
		r.text(left, 12*scale, "INVITATION SAVED", Green)
		r.text(left, 13*scale, "Send this file privately to your friend:", Muted)
		path := strings.TrimPrefix(notice, "Saved to: ")
		for row := 14 * scale; path != "" && row < 21*scale; row++ {
			n := min(len([]rune(path)), width)
			runes := []rune(path)
			r.text(left, row, string(runes[:n]), Bone)
			path = string(runes[n:])
		}
	} else if ready {
		r.text(left, 12*scale, "01 / SEND IT PRIVATELY", Gold)
		r.text(left, 13*scale, "Copy the invitation, or send the saved invitation.txt file.", Bone)
		r.text(left, 15*scale, "02 / YOUR FRIEND JOINS", Gold)
		r.text(left, 16*scale, "They choose Join. Clipboard or paste connects automatically.", Bone)
		r.text(left, 18*scale, "Keep this game running. I reopens your invitation from the fleet.", Muted)
	} else {
		r.text(left+2, 13*scale, "ENTER  Start arranging your fleet", Bone)
		r.text(left+2, 15*scale, "I      Return here to copy your invitation", Muted)
	}
	controls := InvitationControls(sd, ready)
	for _, c := range controls {
		color := Bone
		if c.Key == "c" || c.Key == "s" {
			r.factionPanel(c, Panel, .95)
		}
		if c.Key == hover {
			r.factionPanel(c, 0xffffff, .12)
			color = 0xffffff
		}
		x, y := c.X, c.Y
		if c.Key == "c" || c.Key == "s" {
			x += (c.W - len(c.Label)) / 2
			y += (c.H - 1) / 2
		}
		r.text(x, y, c.Label, color)
	}
	if notice != "" && !strings.HasPrefix(notice, "Saved to: ") {
		color := Cyan
		if strings.HasPrefix(notice, "Couldn") {
			color = Rust
		}
		r.noticeBox(left, 19*scale, width, notice, color)
	}
	footer := controls[0].Y
	r.line(left, (footer-1)*2, r.W-left-1, (footer-1)*2, mix(Muted, Ink, .65))
	const brand = "by TTY.games"
	r.text(r.W-left-len(brand), footer, brand, Muted)
	return r.frame()
}
