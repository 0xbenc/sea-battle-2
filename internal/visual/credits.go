package visual

// RenderCredits shares the title screen's living ocean and restrained palette.
func RenderCredits(sd bool, t float64, idle MenuIdle, hover string) *Frame {
	r := frontOcean(sd, t, idle)
	for i, color := range r.Pixels {
		r.Pixels[i] = mix(color, Ink, .80)
	}
	title, first, gap := 2, 6, 3
	if !sd {
		title, first, gap = 5, 12, 7
	}
	r.center(title, "SEA BATTLE 2", Bone)
	r.center(title+2, "C R E D I T S", Gold)
	credits := [][2]string{
		{"Fearless Leader", "occrahim"},
		{"Lead Programmer and Artist", "0xbenc"},
		{"Network Programming and Playtesting", "basedvik"},
		{"Published by", "Off Court Creations LLC"},
	}
	for i, credit := range credits {
		row := first + i*gap
		r.center(row, credit[0], Gold)
		r.center(row+1, credit[1], Bone)
	}
	r.pageFooter(sd, hover)
	return r.frame()
}
