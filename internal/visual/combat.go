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
	if b.Clip != nil || b.Handoff {
		return nil
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
	controls = append(controls, MenuControl{Key: action, Label: label, X: PlacementGeometry(b.SD).X, Y: fire, W: 18, H: 1})
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
func (r *Raster) combatBoard(b BattlePresentation, g CombatBoard, own, large bool) {
	marks := b.State.Outgoing
	cursor := b.Cursor
	if own {
		marks = b.State.Incoming
		cursor = b.OwnCursor
	}
	accent := Gold
	if b.State.Player == 1 {
		accent = Green
	}
	latest := latestShot(b.State, own)
	for row := 0; row < 10; row++ {
		for col := 0; col < 10; col++ {
			color := RGB(0x132e3a)
			if (row+col)%2 == 1 {
				color = 0x203e49
			}
			if large && (row == cursor.Y || col == cursor.X) {
				color = mix(color, accent, .10)
			}
			mark := marks[row*10+col]
			if mark == game.Miss {
				color = 0x244d5b
			}
			if mark >= game.Hit {
				color = 0x503632
			}
			if mark == game.Sunk {
				color = 0x28343d
			}
			r.rect(g.X+col*g.CW, g.Y+row*g.CH, g.CW, g.CH, color)
		}
	}
	for i := 0; i <= 10; i++ {
		r.line(g.X+i*g.CW, g.Y, g.X+i*g.CW, g.Y+10*g.CH, 0x3f5964)
		r.line(g.X, g.Y+i*g.CH, g.X+10*g.CW, g.Y+i*g.CH, 0x3f5964)
	}
	for i := 0; i < 10; i++ {
		fg, bg := Bone, Ink
		if large && cursor.X == i {
			fg, bg = Ink, accent
		}
		r.solidText(g.X+i*g.CW+g.CW/2, g.Y/2-1, string(rune('A'+i)), fg, bg)
		fg, bg = Bone, Ink
		if large && cursor.Y == i {
			fg, bg = Ink, accent
		}
		r.solidText(g.X-3, (g.Y+i*g.CH+g.CH/2)/2, fmt.Sprintf("%2d", i+1), fg, bg)
	}
	if own {
		l := PlacementLayout{X: g.X, Y: g.Y, CW: g.CW, CH: g.CH}
		for i, p := range b.State.Own {
			if i < 5 && p.Valid() {
				if b.SD && !large {
					r.combatSmallHull(b, g, i, p)
				} else {
					r.placementHull(b, l, i, p, false, true)
				}
			}
		}
	}
	if latest != nil {
		x, y := g.X+latest.At.X*g.CW, g.Y+latest.At.Y*g.CH
		color := Cyan
		if latest.Hit {
			color = 0xffd778
		}
		if latest.Sunk {
			color = 0xa1adb0
		}
		r.line(x, y, x+g.CW, y, color)
		r.line(x, y+g.CH, x+g.CW, y+g.CH, color)
	}
	if large {
		x, y := g.X+cursor.X*g.CW, g.Y+cursor.Y*g.CH
		// Small corner brackets keep the square and its result readable.
		for _, dx := range []int{0, g.CW} {
			for _, dy := range []int{0, g.CH} {
				r.put(x+dx, y+dy, accent)
			}
		}
	}
	for i, mark := range marks {
		if mark == game.Unknown {
			continue
		}
		r.combatImpact(g.X+i%10*g.CW+1, g.Y+i/10*g.CH+1, g.CW-1, g.CH-1, mark, b.T)
	}
}
func (r *Raster) combatButtons(b BattlePresentation) {
	for _, c := range CombatControls(b) {
		if b.Handoff && c.Key == "space" {
			continue
		}
		enabled := true
		if c.Key == "enter" {
			enabled = combatCanFire(b)
		}
		if c.Key == "r" {
			enabled = !b.Pending && !b.State.Rematch[b.State.Player]
		}
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
		r.text(c.X, c.Y, label, color)
	}
}
func (r *Raster) combatHeader(b BattlePresentation, status string, color RGB) {
	r.text(4, 0, status, color)
	team := "A / AMERICAN NAVY"
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
		status, color = "DEFEAT / THE OCEAN HAS NEW MANAGEMENT", Rust
		if b.State.Winner == b.State.Player {
			status, color = "VICTORY / OCEAN LIBERATED", accent
		}
	}
	r.combatHeader(b, status, color)
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
	r.text(smallX, heading, smallLabel, Muted)
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
		col := Muted
		health := shipHealth(b.State, i)
		if health == 0 {
			col = 0x5d6266
		}
		control := MenuControl{X: smallX, Y: row, W: l.Width, H: 1}
		if b.Hover == fmt.Sprintf("fleet%d", i) {
			r.factionPanel(control, 0xffffff, .08)
			col = Bone
		}
		r.text(smallX, row, assets.Names[b.State.Player*5+i], col)
		for j := 0; j < game.Lengths[i]; j++ {
			text, pip := "■", accent
			if j >= health {
				text, pip = "×", Rust
			}
			r.text(smallX+l.Width-5+j, row, text, pip)
		}
	}
	coach := fmt.Sprintf("%s / Click or Enter to fire.  Ripple miss / Fire hit", b.Cursor)
	if b.State.Outgoing[b.Cursor.Index()] != game.Unknown {
		coach = fmt.Sprintf("%s / %s. Choose an unmarked cell.", b.Cursor, shotWord(b.State.Outgoing[b.Cursor.Index()]))
	}
	if b.State.Turn != b.State.Player {
		coach = "Enemy taking aim. You can inspect either chart while waiting."
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
		coach = "Shot transmitted. Awaiting result."
	}
	if !b.State.Connected[0] || !b.State.Connected[1] {
		coach = "Battle paused until both commanders reconnect."
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
			coach = "Rematch requested. Waiting for the other commander."
			if b.Solo {
				coach = "Preparing the next battle..."
			}
		}
	}
	if b.Notice != "" {
		coach = b.Notice
	}
	r.plaque(2, l.Coach, w-4, 1, Ink)
	r.text(4, l.Coach, short(coach, w-8), Bone)
	r.combatButtons(b)
	if b.CombatHelp {
		lines := []string{"FIRE CONTROL", "", "Mouse / arrows / WASD    Aim at hostile waters", "Click / Enter            Fire one shot", "Tab / click small chart  Switch chart focus", "Click a fleet name       Inspect that ship", "Ripple = miss   Fire = hit   Wreck = sunk", "A-J are columns; 1-10 are rows (B4 = column B, row 4).", "", "A hit does not grant an extra shot.", "Only confirmed enemy contacts appear on the chart.", "Space / Enter skips shot cinematics.", "", "Click / ? / Esc close"}
		r.Texts = nil
		width := min(68, w-8)
		left := (w - width) / 2
		top := (h - len(lines) - 2) / 2
		r.plaque(left, top, width, len(lines)+2, Ink)
		for i, line := range lines {
			r.text(left+2, top+1+i, line, Bone)
		}
	}
	return r
}
func RenderCombat(b BattlePresentation) *Frame { return combatRaster(b).frame() }
