package visual

// MenuControl bounds use terminal cells, shared by rendering and mouse input.
type MenuControl struct {
	Key, Label, Detail string
	X, Y, W, H         int
}

func MenuControls(sd bool) []MenuControl {
	w, top, width, height := 160, 42, 108, 2
	footer := 45
	if sd {
		w, top, width, height, footer = 80, 20, 72, 1, 22
	}
	left, column := (w-width)/2, width/3
	controls := []MenuControl{
		{Key: "s", Label: "S  Single player", Detail: "VS COMPUTER"},
		{Key: "h", Label: "H  Host a battle", Detail: "INVITE A PLAYER"},
		{Key: "j", Label: "J  Join by invitation", Detail: "PASTE INVITATION"},
	}
	for i := range controls {
		controls[i].X, controls[i].Y = left+i*column+1, top
		controls[i].W, controls[i].H = column-2, height
	}
	textWidth := len("? Help    T Settings    Q quit")
	x := (w - textWidth) / 2
	for i, label := range []string{"? Help", "T Settings", "Q quit"} {
		offsets := []int{0, 10, 24}
		keys := []string{"?", "t", "q"}
		controls = append(controls, MenuControl{Key: keys[i], Label: label, X: x + offsets[i] - 1, Y: footer, W: len(label) + 2, H: 1})
	}
	return controls
}

func MenuControlAt(sd bool, x, y int) string {
	for _, c := range MenuControls(sd) {
		if x >= c.X && x < c.X+c.W && y >= c.Y && y < c.Y+c.H {
			return c.Key
		}
	}
	return ""
}

// A neutral hover wash brightens only the active control, retaining the ocean.
func (r *Raster) menuControls(notice, hover string) {
	width, height, bottom := 108, 7, 1
	if r.W <= 80 {
		width, height = 72, 5
	}
	left, top := (r.W-width)/2, r.H/2-height-bottom
	for y := top * 2; y < (top+height)*2; y++ {
		for x := left; x < left+width; x++ {
			if (x == left || x == left+width-1) && (y == top*2 || y == (top+height)*2-1) {
				continue
			}
			r.blend(x, y, 0x000000, .45)
		}
	}
	controls := MenuControls(r.W <= 80)
	for _, c := range controls {
		color := Bone
		if c.Detail == "" {
			color = Muted
		}
		if c.Key == hover {
			for y := c.Y * 2; y < (c.Y+c.H)*2; y++ {
				for x := c.X; x < c.X+c.W; x++ {
					r.blend(x, y, 0xffffff, .10)
				}
			}
			color = 0xffffff
		}
		r.text(c.X+1, c.Y, c.Label, color)
		r.text(c.X+1, c.Y, c.Label[:1], 0xffffff)
		if c.H > 1 {
			r.text(c.X+4, c.Y+1, c.Detail, Muted)
		}
	}
	if notice != "" {
		r.center(controls[3].Y-1, short(notice, width-6), Cyan)
	}
}
