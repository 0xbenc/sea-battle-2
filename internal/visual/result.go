package visual

import (
	"fmt"
	"math"
	"sea-battle-2/internal/assets"
	"sea-battle-2/internal/game"
)

func ResultControls(sd bool) []MenuControl {
	y, center := 46, 80
	if sd {
		y, center = 22, 40
	}
	return []MenuControl{
		{Key: "r", Label: "R Rematch", X: center - 34, Y: y, W: 15, H: 1},
		{Key: "tab", Label: "TAB Charts", X: center - 15, Y: y, W: 14, H: 1},
		{Key: "esc", Label: "ESC Menu", X: center + 3, Y: y, W: 13, H: 1},
		{Key: "q", Label: "Q Quit", X: center + 23, Y: y, W: 10, H: 1},
	}
}

// resultShip draws native hull art at the requested scene scale. Wrecks tilt
// through a fixed waterline; surviving ships preserve their native silhouette.
func (r *Raster) resultShip(ship int, sd bool, cx, water, scale, age float64, wreck, scarred bool) {
	size := "HD"
	if sd {
		size = "SD"
	}
	c := assets.Get(fmt.Sprintf("ship-%d-side", ship), size)
	w, h := max(1, int(float64(c.W)*scale)), max(1, int(float64(c.H)*scale))
	x := int(cx) - w/2
	bob := math.Sin(age*.9+float64(ship)) * float64(r.W) / 160
	y := int(water - float64(h)*.87 + bob)
	drop := 0.0
	if wreck {
		drop = .12 + smooth(age/7)*.32
	}
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			n := c.Pixels[(py*c.H/h)*c.W+px*c.W/w]
			if n == 0 {
				continue
			}
			dy := y + py
			if wreck {
				dy += int(float64(h)*drop + float64(px-w/2)*.2)
			}
			if dy > int(water) {
				continue
			}
			color := RGB(assets.Palette[n])
			if wreck {
				color = mix(color, 0x172a32, .48)
			}
			if scarred && (px > w/3 && px < w/2 || px > w*3/4) {
				color = mix(color, 0x302c31, .47)
			}
			r.put(x+px, dy, color)
			if py > h/2 {
				r.blend(x+px+int(math.Sin(age+float64(py)))*2, int(water)+1+(int(water)-dy)/3, color, .16)
			}
		}
	}
	r.wake(int(cx), int(water)+1, w, age, 0x557782)
	if scarred || wreck {
		fx := x + w*3/5
		fy := min(int(water), y+h/2+int(drop*float64(h)))
		s := float64(r.W) / 160
		r.smoke(fx, fy, age, ship >= 5, s*.75)
		r.fire(fx, fy, age, ship >= 5, s*.45)
	}
}

func RenderResult(b BattlePresentation) *Frame {
	w, h := 160, 48
	if b.SD {
		w, h = 80, 24
	}
	r := New(w, h)
	winner := b.State.Winner
	won := winner == b.State.Player
	hp := game.WinnerHullRemaining(b.State)
	dominant := hp >= game.OverwhelmingHP
	t := max(0, b.ResultTime)
	// Four winning tableaux; losing views change camera, weather, and foreground.
	weather := 1
	if dominant {
		weather = 0
	}
	if winner == 1 {
		weather = 3
	}
	r.ocean(t, weather, false)
	tint := RGB(0x513632)
	if winner == 1 {
		tint = 0x34253e
	}
	if !won {
		tint = 0x111c2b
	}
	for i, c := range r.Pixels {
		r.Pixels[i] = mix(c, tint, .22)
	}
	scale := float64(w) / 80
	// Backlit horizon: searchlights after a costly victory, broad occupation
	// beams after a rout. Brood light comes from drifting spores and living arcs.
	accent := Gold
	if winner == 1 {
		accent = Green
	}
	for i := 0; i < 3; i++ {
		base := float64(w) * (float64(i) + 1) / 4
		angle := math.Sin(t*.16+float64(i)*2) * float64(w) * .16
		for y := int(float64(r.H) * .22); y < int(float64(r.H)*.47); y++ {
			p := float64(y) / float64(r.H)
			x := int(base + angle*(.47-p)*3)
			for dx := -int(2 * scale); dx <= int(2*scale); dx++ {
				r.blend(x+dx, y, accent, .06)
			}
		}
	}
	// Choose a representative surviving flagship using already-public sunk IDs.
	hero := 4
	dead := [5]bool{}
	for _, e := range b.State.Events {
		if e.Player != winner && e.Sunk && e.Ship >= 0 && e.Ship < 5 {
			dead[e.Ship] = true
		}
	}
	for hero > 0 && dead[hero] {
		hero--
	}
	if dominant {
		escort := 0
		for i := 0; i < 5 && escort < 3; i++ {
			if dead[i] || i == hero {
				continue
			}
			cx := float64(w) * (.15 + float64(escort)*.35)
			r.resultShip(winner*5+i, b.SD, cx, float64(r.H)*.49, .30, t+float64(i), false, false)
			escort++
		}
		if winner == 0 {
			pass := math.Mod(t*5, 125) - 20
			for i := 0; i < 3; i++ {
				r.idleJet(pass-float64(i)*8, 18+float64(i%2)*3, 1, false, true)
			}
		} else {
			for i := 0; i < 35; i++ {
				x := int(math.Mod(hash(i, 81)*float64(w)+t*scale, float64(w)))
				y := int(float64(r.H)*(.28+hash(i, 82)*.30) + math.Sin(t*.5+float64(i))*2*scale)
				r.blend(x, y, Green, .25+.25*math.Sin(t+float64(i)))
			}
			// Bioluminescent tendrils claim the sea around the victorious brood.
			for i := 0; i < 5; i++ {
				x := float64(w) * (.08 + float64(i)*.21)
				for j := 0; j < 12; j++ {
					r.put(int(x+math.Sin(float64(j)*.35+t*.45)*3*scale), int(float64(r.H)*.64+float64(j)*scale*.45), mix(0x405867, Green, .35))
				}
			}
		}
	}
	if won {
		r.resultShip(winner*5+hero, b.SD, float64(w)*.50, float64(r.H)*.72, 1, t, false, !dominant)
		if !dominant {
			// Wreckage and a rescue light make survival feel costly.
			for i := 0; i < 9; i++ {
				x := int(float64(w) * (.12 + hash(i, 17)*.76))
				y := int(float64(r.H)*(.74+hash(i, 21)*.06) + math.Sin(t+float64(i))*scale)
				r.line(x, y, x+int(2*scale), y, 0x716862)
			}
		}
	} else {
		r.resultShip(winner*5+hero, b.SD, float64(w)*.65, float64(r.H)*.55, .55, t, false, !dominant)
		r.resultShip(b.State.Player*5+4, b.SD, float64(w)*.38, float64(r.H)*.77, .92, t, true, true)
		if dominant { // Multiple wrecks, no rescue: the fleet has been erased.
			r.resultShip(b.State.Player*5+2, b.SD, float64(w)*.84, float64(r.H)*.72, .55, t+2, true, true)
		}
	}
	// Embers/organic motes move slowly through the foreground, below the title.
	for i := 0; i < 14; i++ {
		x := int(hash(i, 91)*float64(w) + math.Sin(t*.3+float64(i))*scale)
		y := int(float64(r.H)*.45 + math.Mod(hash(i, 92)*float64(r.H)*.33-t*.5*scale+1000, float64(r.H)*.33))
		r.blend(x, y, accent, .3)
	}
	word, grade := "VICTORY", "CLOSE VICTORY"
	if dominant {
		grade = "OVERWHELMING VICTORY"
	}
	if !won {
		word = "DEFEAT"
		grade = "NARROW DEFEAT"
		if dominant {
			grade = "OVERWHELMING DEFEAT"
		}
		accent = Rust
	}
	row, sub, quip, stats := 2, 6, 19, 20
	if !b.SD {
		row, sub, quip, stats = 4, 11, 39, 41
	}
	mask := New(w, h)
	mask.bigWord(word, row, accent)
	for i, c := range mask.Pixels {
		if c != 0 {
			x, y := i%w, i/w
			r.rect(x-1, y-1, 3, 3, Ink)
		}
	}
	for i, c := range mask.Pixels {
		if c != 0 {
			r.put(i%w, i/w, c)
		}
	}
	r.center(sub, grade, Bone)
	line := "Victory. Repairs sold separately."
	if b.State.Player == 0 {
		if won && dominant {
			line = "Pest control. Now with a navy."
		}
		if !won {
			line = "They ate the victory speech."
			if dominant {
				line = "Humanity has been reassigned to seafood."
			}
		}
	} else {
		line = "The brood lives. Mostly."
		if won && dominant {
			line = "All-you-can-eat democracy."
		}
		if !won {
			line = "Evolution will be filing an appeal."
			if dominant {
				line = "The food chain has been amended."
			}
		}
	}
	r.center(quip, line, Bone)
	owner := "YOUR"
	if !won {
		owner = "ENEMY"
	}
	if b.Pending || b.State.Rematch[b.State.Player] {
		line = "Waiting for the other commander..."
		if b.Solo {
			line = "Preparing the next battle..."
		}
		r.center(stats, line, Bone)
	} else {
		r.center(stats, fmt.Sprintf("%s FLEET: %d/17 HULL INTACT", owner, hp), Muted)
	}
	for _, c := range ResultControls(b.SD) {
		color := Bone
		if c.Key == "r" && (b.Pending || b.State.Rematch[b.State.Player]) {
			color = Muted
		}
		if b.Hover == c.Key {
			r.factionPanel(c, 0xffffff, .12)
		}
		r.text(c.X, c.Y, c.Label, color)
	}
	return r.frame()
}
