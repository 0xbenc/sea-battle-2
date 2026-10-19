package visual

// A three-step guide keeps the main actions readable at a glance.
func (r *Raster) placementGuide(sd bool, accent RGB) {
	left, width, top, scale := 16, 128, 4, 2
	if sd {
		left, width, top, scale = 4, 72, 2, 1
	}
	r.factionPanel(MenuControl{X: left, Y: top, W: width, H: 20 * scale}, Ink, .96)
	r.text(left+3, top+scale, "PLACE YOUR FLEET", accent)
	r.text(left+3, top+3*scale, "Move with your mouse or arrow keys.", Muted)
	cards := [][3]string{{"1  PICK", "Choose a ship", "1-5 / Tab"}, {"2  TURN", "Rotate it", "Wheel / X"}, {"3  PLACE", "Place on the grid", "Click / Enter"}}
	gap := 2
	cardWidth := (width - 6 - gap*2) / 3
	for i, card := range cards {
		x, y := left+3+i*(cardWidth+gap), top+5*scale
		r.factionPanel(MenuControl{X: x, Y: y, W: cardWidth, H: 5 * scale}, Panel, .95)
		r.text(x+2, y+scale, card[0], accent)
		r.text(x+2, y+2*scale, card[1], Bone)
		r.text(x+2, y+4*scale, card[2], Cyan)
	}
	r.text(left+3, top+11*scale, "R  Shuffle", Bone)
	r.text(left+width/2, top+11*scale, "U  Undo", Bone)
	r.text(left+3, top+13*scale, "Ships can touch. Just don't overlap them.", Muted)
	r.text(left+3, top+14*scale, "Click a ship to move it. Del removes it.", Muted)
	launch := MenuControl{X: left + 3, Y: top + 16*scale, W: width - 6, H: scale}
	r.factionPanel(launch, accent, .18)
	label := "SPACE  Ready to play!"
	r.text(launch.X+(launch.W-len(label))/2, launch.Y, label, Bone)
	r.text(left+3, top+18*scale, "Click anywhere / ? / Esc to close", Muted)
}
