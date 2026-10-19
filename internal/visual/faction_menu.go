package visual

import (
	"sea-battle-2/internal/assets"
	"strings"
)

func FactionControls(sd bool) []MenuControl {
	left, width, top, height, difficulty, footer := 24, 54, 7, 28, 38, 44
	if sd {
		left, width, top, height, difficulty, footer = 4, 34, 7, 7, 16, 21
	}
	controls := []MenuControl{
		{Key: "a", Label: "A  AMERICAN NAVY", X: left, Y: top, W: width, H: height},
		{Key: "b", Label: "B  THE BROOD", X: left + width + 4, Y: top, W: width, H: height},
	}
	center := 80
	if sd {
		center = 40
	}
	for i, label := range []string{"1 Easy", "2 Medium", "3 Hard"} {
		controls = append(controls, MenuControl{Key: string(rune('1' + i)), Label: label, X: center - 24 + i*17, Y: difficulty, W: 14, H: 1})
	}
	controls = append(controls, MenuControl{Key: "esc", Label: "ESC Back", X: center - 13, Y: footer, W: 12, H: 1}, MenuControl{Key: "q", Label: "Q Quit", X: center + 3, Y: footer, W: 10, H: 1})
	return controls
}
func FactionControlAt(sd bool, x, y int) string {
	for _, c := range FactionControls(sd) {
		if x >= c.X && x < c.X+c.W && y >= c.Y && y < c.Y+c.H {
			return c.Key
		}
	}
	return ""
}
func (r *Raster) factionPanel(c MenuControl, tint RGB, alpha float64) {
	for y := c.Y * 2; y < (c.Y+c.H)*2; y++ {
		for x := c.X; x < c.X+c.W; x++ {
			if (x == c.X || x == c.X+c.W-1) && (y == c.Y*2 || y == (c.Y+c.H)*2-1) {
				continue
			}
			r.blend(x, y, tint, alpha)
		}
	}
}

func RenderFactionMenu(sd bool, t float64, difficulty, hover, pressed string, idle MenuIdle) *Frame {
	r := frontOcean(sd, t, idle)
	// Keep the existing ocean framing, subdued enough to read the selection cards.
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			r.blend(x, y, Ink, .42)
		}
	}
	controls := FactionControls(sd)
	r.text(controls[0].X+2, 4, "SINGLE PLAYER / CHOOSE COMMAND", Gold)
	r.text(controls[0].X+2, 5, "Choose your navy. The computer commands the other fleet.", Muted)
	for i, c := range controls[:2] {
		tint, accent := RGB(0x25465b), RGB(0xacc5c3)
		if i == 1 {
			tint, accent = 0x50334f, Green
		}
		r.factionPanel(c, tint, .58)
		if hover == c.Key {
			r.factionPanel(c, 0xffffff, .09)
		}
		if pressed == c.Key {
			r.factionPanel(c, 0xffffff, .2)
		}
		r.text(c.X+2, c.Y+1, c.Label, accent)
		r.text(c.X+2, c.Y+1, c.Label[:1], 0xffffff)
		if !sd {
			ship := 4
			if i == 1 {
				ship = 9
			}
			art := assets.Get("ship-"+string(rune('0'+ship))+"-front", "HD")
			r.asset(art, c.X+(c.W-art.W)/2, (c.Y+4)*2, 0, 0, false)
		}
		line := c.Y + c.H - 4
		if i == 0 {
			r.text(c.X+2, line, "Steel hulls.", Bone)
			r.text(c.X+2, line+1, "Questionable diplomacy.", Muted)
		} else {
			r.text(c.X+2, line, "Organic nastiness.", Bone)
			r.text(c.X+2, line+1, "Some assembly assimilated.", Muted)
		}
		r.text(c.X+2, c.Y+c.H-1, "CLICK / "+strings.ToUpper(c.Key)+"  DEPLOY", Bone)
	}
	row := controls[2].Y
	r.factionPanel(MenuControl{X: controls[0].X, Y: row - 1, W: controls[1].X + controls[1].W - controls[0].X, H: 3}, 0x000000, .3)
	r.center(row-1, "DIFFICULTY", Muted)
	selected := map[string]string{"easy": "1", "medium": "2", "hard": "3"}[difficulty]
	for _, c := range controls[2:] {
		col := Muted
		if c.Key == selected {
			r.factionPanel(c, 0xffffff, .15)
			col = Bone
		}
		if c.Key == hover {
			r.factionPanel(c, 0xffffff, .1)
			col = 0xffffff
		}
		r.text(c.X+2, c.Y, c.Label, col)
	}
	return r.frame()
}
