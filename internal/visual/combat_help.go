package visual

func (r *Raster) combatGuide(sd bool, accent RGB) {
	left, width, top, scale := 16, 128, 4, 2
	if sd {
		left, width, top, scale = 4, 72, 2, 1
	}
	for i, c := range r.Pixels {
		r.Pixels[i] = mix(c, Ink, .7)
	}
	r.Texts = nil
	r.factionPanel(MenuControl{X: left, Y: top, W: width, H: 20 * scale}, Ink, .96)
	r.text(left+3, top+scale, "TAKE YOUR SHOT", accent)
	r.text(left+3, top+3*scale, "One shot each turn. Even when you hit.", Muted)
	cards := [][3]string{
		{"1  AIM", "Pick a square", "Mouse / arrows"},
		{"2  FIRE", "Take your shot", "Click / Enter"},
		{"3  CHECK", "See your fleet", "Tab / small chart"},
	}
	gap := 2
	cardWidth := (width - 6 - gap*2) / 3
	for i, card := range cards {
		x, y := left+3+i*(cardWidth+gap), top+5*scale
		r.factionPanel(MenuControl{X: x, Y: y, W: cardWidth, H: 5 * scale}, Panel, .95)
		r.text(x+2, y+scale, card[0], accent)
		r.text(x+2, y+2*scale, card[1], Bone)
		r.text(x+2, y+4*scale, card[2], Cyan)
	}
	r.chartLegend(left+3, top+11*scale)
	r.text(left+3, top+13*scale, "Ship numbers match your fleet list.", Muted)
	r.text(left+3, top+14*scale, "Click a ship's name to check its damage.", Muted)
	r.text(left+3, top+16*scale, "Space / Enter skips the shot animation.", Bone)
	r.text(left+3, top+18*scale, "Click anywhere / ? / Esc to close", Muted)
}
