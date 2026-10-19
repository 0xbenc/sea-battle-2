package visual

import (
	"fmt"
	"strings"

	"sea-battle-2/internal/assets"
	"sea-battle-2/internal/game"
)

type CombatBoard struct{ X, Y, CW, CH int }

func CombatBoards(sd, inspectOwn bool) (target, own CombatBoard) {
	l := PlacementGeometry(sd)
	large := CombatBoard{l.X, l.Y, l.CW, l.CH}
	small := CombatBoard{110, 18, 4, 3}
	if sd {
		small = CombatBoard{54, 10, 2, 2}
	}
	if inspectOwn {
		return small, large
	}
	return large, small
}
func (b CombatBoard) CellAt(x, row int) (game.Coord, bool) {
	y := row*2 + 1
	if x < b.X || x >= b.X+10*b.CW || y < b.Y || y >= b.Y+10*b.CH {
		return game.Coord{}, false
	}
	return game.Coord{X: (x - b.X) / b.CW, Y: (y - b.Y) / b.CH}, true
}
func CombatControls(b BattlePresentation) []MenuControl {
	if b.Handoff {
		return nil
	}
	if b.Clip != nil {
		row := 46
		if b.SD {
			row = 22
		}
		return []MenuControl{{Key: "space", Label: "SPACE / ENTER Skip", X: 4, Y: row, W: 20, H: 1}}
	}
	if b.State.Phase == "finished" && !b.ResultCharts {
		return ResultControls(b.SD)
	}
	w, fire, footer := 160, 39, 46
	if b.SD {
		w, fire, footer = 80, 20, 23
	}
	controls := []MenuControl{}
	action, label := "enter", "ENTER Fire"
	if b.InspectOwn {
		action, label = "tab", "TAB Targets"
	}
	if b.State.Phase == "finished" {
		action, label = "r", "R Rematch"
	}
	height := 3
	if b.SD {
		height = 1
	}
	controls = append(controls, MenuControl{Key: action, Label: label, X: PlacementGeometry(b.SD).X, Y: fire, W: 18, H: height})
	controls = append(controls, MenuControl{Key: "?", Label: "? Help", X: 4, Y: footer, W: 10, H: 1})
	label = "TAB Fleet"
	if b.State.Phase == "finished" {
		label = "TAB Result"
	}
	if b.InspectOwn && b.State.Phase != "finished" {
		label = "TAB Targets"
	}
	controls = append(controls, MenuControl{Key: "tab", Label: label, X: w/5 + 2, Y: footer, W: 14, H: 1})
	controls = append(controls, MenuControl{Key: "esc", Label: "ESC Menu", X: w*2/5 + 4, Y: footer, W: 12, H: 1})
	if !b.Solo {
		if b.Host {
			controls = append(controls, MenuControl{Key: "i", Label: "I Invite", X: w*3/5 + 6, Y: footer, W: 10, H: 1})
		} else if !b.State.Connected[0] || !b.State.Connected[1] {
			controls = append(controls, MenuControl{Key: "j", Label: "J Rejoin", X: w*3/5 + 6, Y: footer, W: 10, H: 1})
		}
	}
	controls = append(controls, MenuControl{Key: "q", Label: "Q Quit", X: w - 10, Y: footer, W: 8, H: 1})
	return controls
}
func CombatFleetControls(sd bool) []MenuControl {
	l := PlacementGeometry(sd)
	row, step := 27, 2
	if sd {
		row, step = 16, 1
	}
	controls := []MenuControl{}
	for i := 0; i < 5; i++ {
		controls = append(controls, MenuControl{Key: fmt.Sprintf("fleet%d", i), X: l.Side, Y: row + i*step, W: l.Width, H: 1})
	}
	return controls
}
func CombatControlAt(b BattlePresentation, x, y int) string {
	for _, c := range CombatControls(b) {
		if x >= c.X && x < c.X+c.W && y >= c.Y && y < c.Y+c.H {
			return c.Key
		}
	}
	return ""
}
func combatCanFire(b BattlePresentation) bool {
	return b.State.Phase == "battle" && b.State.Turn == b.State.Player && !b.Pending && b.State.Connected[0] && b.State.Connected[1] && b.State.Outgoing[b.Cursor.Index()] == game.Unknown && !b.InspectOwn
}
func shotWord(m game.Mark) string {
	switch m {
	case game.Miss:
		return "MISS"
	case game.Hit:
		return "HIT"
	case game.Sunk:
		return "SUNK"
	}
	return "UNFIRED"
}
func latestShot(v game.View, own bool) *game.Event {
	for i := len(v.Events) - 1; i >= 0; i-- {
		if (v.Events[i].Player != v.Player) == own {
			return &v.Events[i]
		}
	}
	return nil
}
func shotLabel(e *game.Event, prefix string) string {
	if e == nil {
		return ""
	}
	result := "MISS"
	if e.Hit {
		result = "HIT"
	}
	if e.Sunk {
		result = "SUNK"
	}
	return fmt.Sprintf("%s: %s / %s", prefix, e.At, result)
}
func shipHealth(v game.View, index int) int {
	if index >= len(v.Own) || !v.Own[index].Valid() {
		return 0
	}
	health := game.Lengths[index]
	for _, c := range game.Cells(v.Own[index], game.Lengths[index]) {
		if c.Valid() && v.Incoming[c.Index()] >= game.Hit {
			health--
		}
	}
	return health
}

// Fleet charts use solid tiles and redundant symbols at every scale.
// A small tile stays legible instead of squeezing a ship illustration into it.
const (
	chartSea  RGB = 0x102630
	chartShip RGB = 0x398d88
	chartMiss RGB = 0x254759
	chartHit  RGB = 0xb65036
	chartSunk RGB = 0x515568
)

func (r *Raster) combatBoard(b BattlePresentation, g CombatBoard, own, large bool) {
	marks, cursor := b.State.Outgoing, b.Cursor
	if own {
		marks, cursor = b.State.Incoming, b.OwnCursor
	}
	accent := Gold
	if b.State.Player == 1 {
		accent = Green
	}
	var hull [100]int
	if own {
		for i, p := range b.State.Own {
			if i >= 5 || !p.Valid() {
				continue
			}
			for _, cell := range game.Cells(p, game.Lengths[i]) {
				if cell.Valid() {
					hull[cell.Index()] = i + 1
				}
			}
		}
	}
	compact := g.CH == 2
	for row := 0; row < 10; row++ {
		for col := 0; col < 10; col++ {
			index := row*10 + col
			color := chartSea
			if (row+col)%2 == 1 {
				color = 0x142d36
			}
			if large && (row == cursor.Y || col == cursor.X) {
				color = mix(color, accent, .09)
			}
			symbol := ""
			if hull[index] > 0 {
				color = chartShip
				symbol = fmt.Sprint(hull[index])
			}
			switch marks[index] {
			case game.Miss:
				color, symbol = chartMiss, "·"
			case game.Hit:
				color, symbol = chartHit, "X"
			case game.Sunk:
				color, symbol = chartSunk, "#"
			}
			x, y := g.X+col*g.CW, g.Y+row*g.CH
			r.rect(x, y, g.CW, g.CH, 0x223a44)
			inset := 1
			if compact {
				inset = 0
			}
			r.rect(x+inset, y+inset, g.CW-inset, g.CH-inset, color)
			if symbol != "" {
				tx, ty := x+g.CW/2, (y+g.CH/2)/2
				if compact {
					tx = x
				}
				r.solidText(tx, ty, symbol, 0xf4f2de, color)
			}
		}
	}
	for i := 0; i < 10; i++ {
		fg, bg := Muted, Ink
		if large && cursor.X == i {
			fg, bg = Ink, accent
		}
		r.solidText(g.X+i*g.CW+g.CW/2, g.Y/2-1, string(rune('A'+i)), fg, bg)
		fg, bg = Muted, Ink
		if large && cursor.Y == i {
			fg, bg = Ink, accent
		}
		r.solidText(g.X-3, (g.Y+i*g.CH+g.CH/2)/2, fmt.Sprintf("%2d", i+1), fg, bg)
	}
	if large {
		x, y := g.X+cursor.X*g.CW, g.Y+cursor.Y*g.CH
		for _, dx := range []int{0, g.CW - 1} {
			for _, dy := range []int{0, g.CH - 1} {
				r.put(x+dx, y+dy, accent)
			}
		}
	}
}

func (r *Raster) chartLegend(x, row int) {
	for _, item := range []struct {
		label string
		color RGB
	}{{"1 Ship", chartShip}, {"· Miss", chartMiss}, {"X Hit", chartHit}, {"# Sunk", chartSunk}} {
		// All legend symbols are one rune; don't split the UTF-8 miss marker.
		runes := []rune(item.label)
		r.solidText(x, row, string(runes[0]), Bone, item.color)
		r.text(x+1, row, string(runes[1:]), Muted)
		x += len(runes) + 3
	}
}

// CombatControlEnabled is shared by drawing and press feedback.
func CombatControlEnabled(b BattlePresentation, key string) bool {
	if key == "enter" {
		return combatCanFire(b)
	}
	if key == "r" {
		return !b.Pending && !b.State.Rematch[b.State.Player]
	}
	return true
}
func (r *Raster) combatButtons(b BattlePresentation) {
	for _, c := range CombatControls(b) {
		if b.Handoff && c.Key == "space" {
			continue
		}
		enabled := CombatControlEnabled(b, c.Key)
		color := Muted
		if enabled {
			color = Bone
		}
		if c.Key == "enter" || c.Key == "r" {
			tint := RGB(0x263c43)
			if enabled {
				tint = Gold
				if b.State.Player == 1 {
					tint = Green
				}
			}
			r.factionPanel(c, tint, .22)
		}
		if enabled && b.Hover == c.Key {
			r.factionPanel(c, 0xffffff, .12)
			color = 0xffffff
		}
		label := c.Label
		if c.Key == "enter" {
			label += " " + b.Cursor.String()
		}
		if c.Key == "r" && !enabled {
			label = "REMATCH REQUESTED"
		}
		y := c.Y + (c.H-1)/2
		r.text(c.X, y, label, color)
	}
}
func (r *Raster) combatHeader(b BattlePresentation, status string, color RGB) {
	r.rect(0, 0, r.W, 4, Panel)
	r.rect(2, 0, 1, 2, color)
	r.text(4, 0, status, color)
	team := "A / UNITED STATES NAVY"
	if b.State.Player == 1 {
		team = "B / THE BROOD"
	}
	if b.Solo {
		team += " / SOLO " + strings.ToUpper(b.Difficulty)
	}
	r.text(4, 1, team+" / BATTLE", Muted)
	counter := fmt.Sprintf("SHOT %03d", len(b.State.Events)+1)
	if b.State.Phase == "finished" {
		counter = fmt.Sprintf("MATCH %d", b.State.Match)
	}
	r.text(r.W-14, 0, counter, Muted)
}
func combatRaster(b BattlePresentation) *Raster {
	w, h := 160, 48
	if b.SD {
		w, h = 80, 24
	}
	r := New(w, h)
	r.ocean(b.T, 0, true)
	for i, c := range r.Pixels {
		r.Pixels[i] = mix(c, Ink, .87)
	}
	accent := Gold
	if b.State.Player == 1 {
		accent = Green
	}
	status, color := "ENEMY TURN", Muted
	if b.State.Turn == b.State.Player {
		status, color = "YOUR TURN", accent
	}
	if b.Pending {
		status = "SENDING SHOT"
	}
	if !b.State.Connected[0] || !b.State.Connected[1] {
		status, color = "LINK LOST / RECONNECTING", Rust
	}
	if b.State.Phase == "finished" {
		status, color = "DEFEAT", Rust
		if b.State.Winner == b.State.Player {
			status, color = "VICTORY", accent
		}
	}
	r.combatHeader(b, status, color)
	if !b.Solo && b.State.Phase != "finished" && (!b.State.Connected[0] || !b.State.Connected[1]) {
		r.loadingDots(r.W-24, 0, b.T)
	}
	target, own := CombatBoards(b.SD, b.InspectOwn)
	r.combatBoard(b, target, false, !b.InspectOwn)
	r.combatBoard(b, own, true, b.InspectOwn)
	l := PlacementGeometry(b.SD)
	heading := 4
	if b.SD {
		heading = 2
	}
	smallX := l.Side
	largeLabel, smallLabel := "HOSTILE WATERS", "FRIENDLY WATERS"
	if b.InspectOwn {
		largeLabel, smallLabel = smallLabel, largeLabel
	}
	r.text(l.X, heading, largeLabel, accent)
	r.text(smallX, heading, smallLabel, accent)
	incoming, outgoing := latestShot(b.State, true), latestShot(b.State, false)
	largeEvent, smallEvent := outgoing, incoming
	prefix := "ENEMY"
	if b.InspectOwn {
		largeEvent, smallEvent = incoming, outgoing
		prefix = "YOU"
	}
	if !b.SD {
		r.text(l.X, heading+1, shotLabel(largeEvent, "LAST"), Muted)
		r.text(smallX, heading+2, shotLabel(smallEvent, prefix), Muted)
	} else {
		r.text(l.X+19, 20, shotLabel(largeEvent, "LAST"), Muted)
		r.text(smallX, heading+1, shotLabel(smallEvent, prefix), Muted)
	}
	// Compact damage rack: each pip is one hull cell, not a generic progress bar.
	fleetRow, step := 27, 2
	if b.SD {
		fleetRow, step = 16, 1
	}
	alive := 0
	for i := 0; i < 5; i++ {
		if shipHealth(b.State, i) > 0 {
			alive++
		}
	}
	knownSunk := map[int]bool{}
	for _, e := range b.State.Events {
		if e.Player == b.State.Player && e.Sunk && e.Ship >= 0 {
			knownSunk[e.Ship] = true
		}
	}
	enemyLeft := 5 - len(knownSunk)
	largeLeft, smallLeft := enemyLeft, alive
	if b.InspectOwn {
		largeLeft, smallLeft = alive, enemyLeft
	}
	r.text(smallX+l.Width-3, heading, fmt.Sprintf("%d/5", smallLeft), Muted)
	r.text(l.X+10*l.CW-3, heading, fmt.Sprintf("%d/5", largeLeft), Muted)
	if !b.SD {
		r.text(smallX, fleetRow-2, "YOUR FLEET", Muted)
	}
	for i := 0; i < 5; i++ {
		row := fleetRow + i*step
		col := Bone
		health := shipHealth(b.State, i)
		if health == 0 {
			col = 0x5d6266
		}
		control := MenuControl{X: smallX, Y: row, W: l.Width, H: 1}
		r.factionPanel(control, Panel, .9)
		if b.InspectOwn && i < len(b.State.Own) && b.State.Own[i].Valid() {
			for _, cell := range game.Cells(b.State.Own[i], game.Lengths[i]) {
				if cell == b.OwnCursor {
					r.factionPanel(control, Cyan, .15)
					break
				}
			}
		}
		if b.Hover == fmt.Sprintf("fleet%d", i) {
			r.factionPanel(control, 0xffffff, .08)
			col = Bone
		}
		r.text(smallX, row, fmt.Sprintf("%d %s", i+1, assets.Names[b.State.Player*5+i]), col)
		if health == 0 {
			r.text(smallX+l.Width-5, row, "SUNK", Muted)
			continue
		}
		for j := 0; j < game.Lengths[i]; j++ {
			text, pip := "■", chartShip
			if j >= health {
				text, pip = "×", Rust
			}
			r.text(smallX+l.Width-5+j, row, text, pip)
		}
	}
	coach := fmt.Sprintf("%s / Click or Enter to fire.", b.Cursor)
	if b.State.Outgoing[b.Cursor.Index()] != game.Unknown {
		coach = fmt.Sprintf("%s / %s. Choose an unmarked cell.", b.Cursor, shotWord(b.State.Outgoing[b.Cursor.Index()]))
	}
	if b.State.Turn != b.State.Player {
		coach = "Your opponent is aiming. You can check your fleet while you wait."
	}
	if b.InspectOwn {
		coach = fmt.Sprintf("YOUR WATERS / %s / %s. Tab returns to targeting.", b.OwnCursor, shotWord(b.State.Incoming[b.OwnCursor.Index()]))
		if b.State.Incoming[b.OwnCursor.Index()] == game.Unknown {
			coach = fmt.Sprintf("YOUR WATERS / %s. Tab returns to targeting.", b.OwnCursor)
		}
	}
	if b.InspectOwn {
		for i, p := range b.State.Own {
			if i >= 5 || !p.Valid() {
				continue
			}
			for _, cell := range game.Cells(p, game.Lengths[i]) {
				if cell == b.OwnCursor {
					coach = fmt.Sprintf("%s / %d of %d intact / %s. Tab targets.", assets.Names[b.State.Player*5+i], shipHealth(b.State, i), game.Lengths[i], b.OwnCursor)
				}
			}
		}
	}
	if b.Pending {
		coach = "Sending your shot…"
	}
	if !b.State.Connected[0] || !b.State.Connected[1] {
		coach = "Battle paused. Reconnecting…"
	}
	if b.State.Phase == "finished" {
		shots, hits := 0, 0
		for _, e := range b.State.Events {
			if e.Player == b.State.Player {
				shots++
				if e.Hit {
					hits++
				}
			}
		}
		accuracy := 0
		if shots > 0 {
			accuracy = hits * 100 / shots
		}
		coach = fmt.Sprintf("%d shots / %d hits / %d%% accuracy. Inspect the charts or play again.", shots, hits, accuracy)
		if b.State.Rematch[b.State.Player] || b.Pending {
			coach = "Rematch requested. Waiting for your opponent."
			if b.Solo {
				coach = "Preparing the next battle…"
			}
		}
	}
	if b.Notice != "" {
		coach = b.Notice
	}
	legendRow := 38
	if b.SD {
		legendRow = 19
	}
	r.chartLegend(l.X, legendRow)
	hintRow := 44
	if b.SD {
		hintRow = 22
	}
	hint := "Click the small chart to switch views. Numbers match your fleet."
	if !b.Solo && b.State.Phase != "finished" && (!b.State.Connected[0] || !b.State.Connected[1]) {
		hint = "J  Paste a fresh invitation if reconnecting takes too long."
		if b.Host {
			hint = "I  Share a fresh invitation if your friend cannot reconnect."
		}
	}
	r.text(4, hintRow, hint, Muted)
	r.rect(2, l.Coach*2, w-4, 2, Panel)
	r.rect(2, l.Coach*2, 1, 2, color)
	r.text(4, l.Coach, short(coach, w-8), Bone)
	r.combatButtons(b)
	if b.CombatHelp {
		r.combatGuide(b.SD, accent)
	}
	return r
}
func RenderCombat(b BattlePresentation) *Frame { return combatRaster(b).frame() }
