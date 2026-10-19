package visual

import (
	"fmt"
	"math"
	"sea-battle-2/internal/assets"
	"sea-battle-2/internal/game"
)

func ShotDuration(e game.Event) float64 {
	if e.Sunk {
		return 6
	}
	if e.Hit {
		return 3.5
	}
	return 2
}

// Choose once from the shot identity, not the animation clock or viewer.
// Both players get the same take; redraws and resize never reroll it.
func shotTake(match int, e game.Event) int {
	n := uint64(match)*0x9e3779b97f4a7c15 + uint64(e.Seq+1)*0xbf58476d1ce4e5b9 + uint64(e.At.Index()+1)*97 + uint64(e.Player)*53
	n ^= n >> 30
	n *= 0xbf58476d1ce4e5b9
	n ^= n >> 27
	n *= 0x94d049bb133111eb
	n ^= n >> 31
	return int(n % 3)
}

func (r *Raster) battleClip(b BattlePresentation, st State) {
	e := *b.Clip
	take := shotTake(b.State.Match, e)
	age := b.ClipTime
	st.T = age
	st.TargetX, st.TargetY = e.At.X, e.At.Y
	scale := float64(r.W) / 160
	weather := []int{0, 1, 2}[take]
	if e.Player == 0 && take == 2 {
		weather = 3
	}
	knownHull := e.Ship >= 0 && e.Ship < 5 && (e.Sunk || e.Player != b.State.Player)
	if knownHull {
		camera := []string{"SIDE", "FRONT", "SIDE"}[take]
		if !e.Sunk && take == 2 {
			camera = "TOP"
		}
		scene := Scene{Kind: Hit, Ship: (1-e.Player)*5 + e.Ship, Team: e.Player, Duration: 9}
		// Start near impact so a short clip never spends half its time waiting.
		st.T = age + 1.15
		if e.Sunk {
			scene.Kind = Sink
			scene.Duration = 13
			st.T = age*1.8 + 1.15
		}
		r.cinemaScene(st, scene, weather, camera, false)
	} else {
		r.ocean(age, weather, false)
		x := int(float64(r.W) * []float64{.38, .62, .5}[take])
		y := int(float64(r.H) * []float64{.64, .60, .69}[take])
		// A short incoming streak leads directly to the splash or impact.
		approach := .25
		if age < approach {
			p := 1 - age/approach
			direction := []float64{-1, 1, -1}[take]
			px := x + int(direction*p*float64(r.W)*.45)
			py := y - int(p*float64(r.H)*.45)
			color := Gold
			if e.Player == 1 {
				color = Green
			}
			r.line(px, py, px+int(direction*6*scale), py-int(3*scale), color)
		}
		impactAge := age - approach
		if impactAge >= 0 {
			if e.Hit {
				r.explosion(x, y, impactAge, e.Player == 1, scale*[]float64{1, 1.2, .9}[take])
				r.burst(x, y, impactAge, st.Size(), e.Player == 1)
				r.smoke(x, y, impactAge, e.Player == 1, scale*.7)
			} else {
				r.splash(x, y, impactAge, scale*[]float64{1.1, .85, 1.3}[take])
				if take == 1 {
					r.splash(x+int(10*scale), y+int(2*scale), impactAge-.18, scale*.5)
				}
			}
		}
		r.rain(age, weather)
	}
	// Fade the upper and lower image into the same dark chrome as the charts.
	for y := 0; y < r.H; y++ {
		shade := 0.0
		if y < r.H/4 {
			shade = .65 * (1 - float64(y)/float64(r.H/4))
		}
		if y > r.H*3/4 {
			shade = .7 * float64(y-r.H*3/4) / float64(r.H/4)
		}
		for x := 0; x < r.W; x++ {
			r.blend(x, y, Ink, shade)
		}
	}
	word, color := "MISS", Cyan
	caption := []string{"All water.", "Wide of the mark.", "Not this square."}[take]
	incoming := e.Player != b.State.Player
	if incoming {
		caption = []string{"Still afloat.", "They missed us.", "Nothing but water."}[take]
	}
	if e.Hit {
		word, color = "HIT", Gold
		caption = []string{"Direct hit.", "Found one.", "Right on target."}[take]
		if incoming {
			color = Rust
			caption = []string{"Hull hit.", "That one landed.", "We're taking damage."}[take]
		}
	}
	if e.Sunk {
		word, color = "SUNK", Gold
		caption = []string{"Down it goes.", "One less ship.", "That's one gone."}[take]
		if incoming {
			color = Rust
			caption = []string{"Ship lost.", "Going under.", "We lost one."}[take]
		}
		if !incoming && e.Player == 0 && take == 1 {
			caption = "One less mouth."
		}
	}
	row := 4
	if !b.SD {
		row = 7
	}
	mask := New(r.W, r.H/2)
	mask.bigWord(word, row, color)
	for i, c := range mask.Pixels {
		if c != 0 {
			r.rect(i%r.W-1, i/r.W-1, 3, 3, Ink)
		}
	}
	for i, c := range mask.Pixels {
		if c != 0 {
			r.put(i%r.W, i/r.W, c)
		}
	}
	source := "YOUR SHOT"
	if incoming {
		source = "INCOMING"
	}
	r.rect(0, 0, r.W, 4, Panel)
	r.text(4, 0, fmt.Sprintf("%s / %s", source, e.At), color)
	r.text(r.W-14, 0, fmt.Sprintf("SHOT %03d", e.Seq), Muted)
	r.text(4, 1, caption, Bone)
	if e.Sunk && knownHull {
		nameRow := 9
		if !b.SD {
			nameRow = 15
		}
		r.center(nameRow, assets.Names[(1-e.Player)*5+e.Ship], Bone)
	}
	footer := r.H/2 - 2
	if b.Hover == "space" {
		r.factionPanel(MenuControl{X: 4, Y: footer, W: 20, H: 1}, 0xffffff, .12)
	}
	r.text(4, footer, "SPACE / ENTER Skip", Bone)
	const brand = "by TTY.games"
	r.text(r.W-4-len(brand), footer, brand, Muted)
	// Subtle timeline communicates the length of the take, without a countdown.
	progress := math.Min(1, math.Max(0, age/ShotDuration(e)))
	r.rect(4, (footer-1)*2, r.W-8, 1, Panel)
	r.rect(4, (footer-1)*2, int(float64(r.W-8)*progress), 1, mix(color, Ink, .35))
}
