package visual

import (
	"fmt"
	"strings"

	"sea-battle-2/internal/assets"
	"sea-battle-2/internal/game"
)

// Board geometry is in half-pixels; controls and pointer coordinates use cells.
type PlacementLayout struct{ X, Y, CW, CH, Side, Width, Top, Step, Coach, Footer int }

func PlacementGeometry(sd bool) PlacementLayout {
	if sd {
		return PlacementLayout{5, 8, 4, 3, 49, 28, 5, 2, 21, 23}
	}
	return PlacementLayout{12, 14, 8, 6, 104, 44, 10, 4, 42, 46}
}
func PlacementCellAt(sd bool, x, row int) (game.Coord, bool) {
	l := PlacementGeometry(sd)
	y := row*2 + 1
	if x < l.X || x >= l.X+10*l.CW || y < l.Y || y >= l.Y+10*l.CH {
		return game.Coord{}, false
	}
	return game.Coord{X: (x - l.X) / l.CW, Y: (y - l.Y) / l.CH}, true
}
func PlacementControls(sd bool) []MenuControl {
	l := PlacementGeometry(sd)
	controls := []MenuControl{}
	for i := 0; i < 5; i++ {
		controls = append(controls, MenuControl{Key: fmt.Sprint(i + 1), X: l.Side, Y: l.Top + i*l.Step, W: l.Width, H: max(2, l.Step-1)})
	}
	row := l.Top + 5*l.Step + 1
	for i, item := range []struct{ key, label string }{{"x", "X Rotate"}, {"backspace", "Del Remove"}, {"r", "R Shuffle"}, {"u", "U Undo"}} {
		controls = append(controls, MenuControl{Key: item.key, Label: item.label, X: l.Side + (i%2)*(l.Width/2), Y: row + (i/2)*2, W: l.Width/2 - 1, H: 1})
	}
	controls = append(controls, MenuControl{Key: "space", Label: "SPACE Launch", X: l.Side, Y: row + 4, W: l.Width, H: 1})
	controls = append(controls, MenuControl{Key: "?", Label: "? Help", X: 4, Y: l.Footer, W: 10, H: 1}, MenuControl{Key: "esc", Label: "ESC Menu", X: 18, Y: l.Footer, W: 12, H: 1}, MenuControl{Key: "q", Label: "Q Quit", X: 34, Y: l.Footer, W: 10, H: 1})
	return controls
}
func PlacementControlAt(sd bool, x, y int) string {
	for _, c := range PlacementControls(sd) {
		if x >= c.X && x < c.X+c.W && y >= c.Y && y < c.Y+c.H {
			return c.Key
		}
	}
	return ""
}
func PlacementComplete(fleet []game.Placement) bool { return game.ValidateFleet(fleet) == nil }
func PlacementDirty(b BattlePresentation) bool {
	if b.Selected < 0 || b.Selected >= len(b.Fleet) || !b.Fleet[b.Selected].Valid() {
		return false
	}
	return b.Fleet[b.Selected] != (game.Placement{Coord: b.Cursor, Horizontal: b.Horizontal})
}
func (r *Raster) placementHull(b BattlePresentation, l PlacementLayout, index int, p game.Placement, ghost bool, legal bool) {
	size := "board-"
	if p.Horizontal {
		size = "board-horizontal-"
	}
	if b.SD {
		size += "SD"
	} else {
		size += "HD"
	}
	canvas := assets.Get(fmt.Sprintf("ship-%d-top", b.State.Player*5+index), size)
	source := New(canvas.W, (canvas.H+1)/2)
	source.asset(canvas, 0, 0, 0, 0, false)
	width, height := l.CW-1, game.Lengths[index]*l.CH-1
	if p.Horizontal {
		width, height = game.Lengths[index]*l.CW-1, l.CH-1
	}
	for yy := 0; yy < height; yy++ {
		for xx := 0; xx < width; xx++ {
			color := source.Pixels[(yy*canvas.H/height)*source.W+xx*canvas.W/width]
			if color == 0 {
				continue
			}
			x, y := l.X+p.X*l.CW+1+xx, l.Y+p.Y*l.CH+1+yy
			if x < l.X || x >= l.X+10*l.CW || y < l.Y || y >= l.Y+10*l.CH {
				continue
			}
			if ghost {
				tint := Green
				if !legal {
					tint = Rust
				}
				color = mix(color, tint, .45)
			}
			r.put(x, y, color)
		}
	}
}
func renderPlacement(b BattlePresentation) *Frame {
	w, h := 160, 48
	if b.SD {
		w, h = 80, 24
	}
	r := New(w, h)
	r.ocean(b.T, 0, true)
	l := PlacementGeometry(b.SD)
	accent, tint := Gold, RGB(0x1c3543)
	if b.State.Player == 1 {
		accent, tint = Green, 0x352c40
	}
	locked := b.State.Ready[b.State.Player]
	fleet := b.Fleet
	if locked {
		fleet = b.State.Own
	}
	active := b.Selected >= 0 && b.Selected < 5 && !locked && !b.Pending
	draft := game.Placement{Coord: b.Cursor, Horizontal: b.Horizontal}
	legal := active && game.CanPlace(fleet, b.Selected, draft)
	r.text(4, 0, "DEPLOY YOUR FLEET", Bone)
	team := "A / AMERICAN NAVY"
	if b.State.Player == 1 {
		team = "B / THE BROOD"
	}
	if b.Solo {
		team += " / SOLO " + strings.ToUpper(b.Difficulty)
	}
	r.text(4, 1, team, accent)
	r.text(l.X, l.Y/2-2, "YOUR WATERS", Muted)
	for row := 0; row < 10; row++ {
		for col := 0; col < 10; col++ {
			bg := RGB(0x18323e)
			if (row+col)%2 == 1 {
				bg = 0x203e49
			}
			if b.State.Player == 1 {
				bg = mix(bg, tint, .3)
			}
			if active && (row == b.Cursor.Y || col == b.Cursor.X) {
				bg = mix(bg, accent, .09)
			}
			r.rect(l.X+col*l.CW, l.Y+row*l.CH, l.CW, l.CH, bg)
		}
	}
	for i := 0; i <= 10; i++ {
		r.line(l.X+i*l.CW, l.Y, l.X+i*l.CW, l.Y+10*l.CH, 0x45616a)
		r.line(l.X, l.Y+i*l.CH, l.X+10*l.CW, l.Y+i*l.CH, 0x45616a)
	}
	for i := 0; i < 10; i++ {
		fg, bg := Bone, Ink
		if active && b.Cursor.X == i {
			fg, bg = Ink, accent
		}
		r.solidText(l.X+i*l.CW+l.CW/2, l.Y/2-1, string(rune('A'+i)), fg, bg)
		fg, bg = Bone, Ink
		if active && b.Cursor.Y == i {
			fg, bg = Ink, accent
		}
		r.solidText(l.X-3, (l.Y+i*l.CH+l.CH/2)/2, fmt.Sprintf("%2d", i+1), fg, bg)
	}
	for i, p := range fleet {
		if i < 5 && p.Valid() && !(active && i == b.Selected) {
			r.placementHull(b, l, i, p, false, true)
		}
	}
	if active {
		color := Green
		if !legal {
			color = Rust
		}
		for _, c := range game.Cells(draft, game.Lengths[b.Selected]) {
			if !c.Valid() {
				continue
			}
			for y := 1; y < l.CH; y++ {
				for x := 1; x < l.CW; x++ {
					r.blend(l.X+c.X*l.CW+x, l.Y+c.Y*l.CH+y, color, .2)
				}
			}
		}
		r.placementHull(b, l, b.Selected, draft, true, legal)
		x, y := l.X+b.Cursor.X*l.CW, l.Y+b.Cursor.Y*l.CH
		r.line(x, y, x+l.CW, y, color)
		r.line(x, y+l.CH, x+l.CW, y+l.CH, color)
	}
	count := 0
	for _, p := range fleet {
		if p.Valid() {
			count++
		}
	}
	r.factionPanel(MenuControl{X: l.Side - 1, Y: l.Top - 2, W: l.Width + 2, H: 5*l.Step + 2}, tint, .75)
	r.text(l.Side, l.Top-2, fmt.Sprintf("FLEET  %d / 5", count), accent)
	controls := PlacementControls(b.SD)
	for i, c := range controls[:5] {
		chosen := active && b.Selected == i
		if chosen {
			r.factionPanel(MenuControl{X: c.X, Y: c.Y, W: c.W, H: 2}, 0xffffff, .1)
		}
		if b.Hover == c.Key && !locked {
			r.factionPanel(MenuControl{X: c.X, Y: c.Y, W: c.W, H: 2}, 0xffffff, .06)
		}
		col := Muted
		if chosen {
			col = Bone
		}
		r.text(c.X, c.Y, fmt.Sprintf("%d %s", i+1, assets.Names[b.State.Player*5+i]), col)
		pips := strings.Repeat("■", game.Lengths[i])
		status := ""
		if len(fleet) > i && fleet[i].Valid() {
			status = "  SET"
		}
		r.text(c.X+2, c.Y+1, pips+status, accent)
	}
	complete := PlacementComplete(fleet) && !PlacementDirty(b)
	for _, c := range controls[5:] {
		enabled := !locked && !b.Pending
		if c.Key == "space" {
			enabled = enabled && complete && b.State.Connected[b.State.Player]
		}
		if c.Key == "x" || c.Key == "backspace" {
			enabled = enabled && active
		}
		if c.Key == "backspace" && active {
			enabled = enabled && fleet[b.Selected].Valid()
		}
		if c.Key == "u" {
			enabled = enabled && b.CanUndo
		}
		if c.Key == "?" || c.Key == "q" || c.Key == "esc" {
			enabled = true
		}
		col := RGB(0x52666c)
		if enabled {
			col = Bone
		}
		if c.Key == "space" && enabled {
			r.factionPanel(c, accent, .18)
		}
		if b.Hover == c.Key && enabled {
			r.factionPanel(c, 0xffffff, .1)
		}
		label := c.Label
		if c.Key == "space" && locked {
			label = "FLEET LOCKED"
		}
		if c.Key == "space" && b.Pending {
			label = "LAUNCHING..."
		}
		r.text(c.X, c.Y, label, col)
	}
	coach := "Place five ships, then launch."
	if active {
		coach = fmt.Sprintf("%s at %s  /  Click or Enter to place.", assets.Names[b.State.Player*5+b.Selected], b.Cursor)
		if !legal {
			coach = "Blocked. Move clear of other ships or rotate with X."
		}
		if PlacementDirty(b) {
			coach = "Click or Enter to move this ship. Esc cancels the move."
			if !legal {
				coach = "Move blocked. Find clear water; Esc cancels."
			}
		}
	} else if complete {
		coach = "Fleet ready. Click a ship to adjust it, or launch."
	}
	if locked {
		coach = "Fleet locked. Waiting for the other commander."
	}
	if b.Pending {
		coach = "Launching fleet..."
	}
	if !b.State.Connected[b.State.Player] {
		coach = "Waiting for connection. You can keep arranging your fleet."
	}
	if b.Notice != "" && !b.Pending && !locked {
		coach = b.Notice
	}
	r.plaque(2, l.Coach, w-4, 1, Ink)
	r.text(4, l.Coach, short(coach, w-8), Bone)
	if b.PlacementHelp {
		r.Texts = nil
		lines := []string{"PLACEMENT CONTROLS", "", "Mouse / arrows / WASD   Move the preview", "Click / Enter           Place the selected ship", "Right click / X         Rotate", "1-5 / Tab               Select a ship", "Delete                  Remove selected ship", "R shuffle    U undo    Space launch", "", "Ships may touch, but cannot overlap.", "Click a placed ship to move it.", "", "Click / ? / Esc close"}
		width := min(68, w-8)
		left := (w - width) / 2
		top := (h - len(lines) - 2) / 2
		r.plaque(left, top, width, len(lines)+2, Ink)
		for i, line := range lines {
			r.text(left+2, top+1+i, line, Bone)
		}
	}
	return r.frame()
}
