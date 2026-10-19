package visual

import "math"

func ConnectingControls(sd bool) []MenuControl {
	controls := FooterControls(sd)
	controls[0].Label = "ESC Cancel"
	controls[0].W = 12
	return controls
}
func ConnectingControlAt(sd bool, x, y int) string {
	for _, c := range ConnectingControls(sd) {
		if x >= c.X && x < c.X+c.W && y >= c.Y && y < c.Y+c.H {
			return c.Key
		}
	}
	return ""
}
func (r *Raster) loadingDots(x, row int, t float64) {
	for i := 0; i < 3; i++ {
		glow := (math.Sin(t*4-float64(i)*.9) + 1) / 2
		r.text(x+i*2, row, "•", mix(Panel, Cyan, .25+.75*glow))
	}
}
func RenderConnecting(sd bool, t, progressTime float64, idle MenuIdle, status, notice, hover string) *Frame {
	r := frontOcean(sd, t, idle)
	for i, c := range r.Pixels {
		r.Pixels[i] = mix(c, Ink, .86)
	}
	left, width, scale := 16, 128, 2
	if sd {
		left, width, scale = 4, 72, 1
	}
	r.text(left, 2*scale, "CONNECTING", Gold)
	r.factionPanel(MenuControl{X: left, Y: 8 * scale, W: width, H: 3 * scale}, Panel, .85)
	if status == "" {
		status = "Establishing a secure connection…"
	}
	r.text(left+2, 9*scale, short(status, width-14), Bone)
	r.loadingDots(left+width-8, 9*scale, progressTime)
	r.text(left+2, 13*scale, "Keep your friend's game running. We'll take it from here.", Muted)
	if notice != "" {
		r.text(left, 18*scale, short(notice, width), Cyan)
	}
	r.controlFooter(ConnectingControls(sd), hover)
	return r.frame()
}
