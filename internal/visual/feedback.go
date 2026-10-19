package visual

// PressFeedback is a short, local accent that fades without delaying actions.
func PressFeedback(f *Frame, c MenuControl, age float64) {
	if age < 0 || age >= .24 {
		return
	}
	strength := 1 - smooth(age/.24)
	for y := max(0, c.Y); y < min(f.H, c.Y+c.H); y++ {
		for x := max(0, c.X); x < min(f.W, c.X+c.W); x++ {
			cell := &f.Cells[y*f.W+x]
			cell.BG = mix(cell.BG, Cyan, .24*strength)
			cell.FG = mix(cell.FG, 0xffffff, .3*strength)
		}
	}
}
