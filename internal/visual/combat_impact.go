package visual

import (
	"math"
	"sea-battle-2/internal/game"
)

// Every impact stays inside its cell. Hand-drawn small variants preserve the
// silhouette when a chart has only two half-pixel rows per square.
func (r *Raster) combatImpact(x, y, w, h int, mark game.Mark, t float64) {
	palette := map[rune]RGB{
		'~': 0x70c3d0, 'o': 0x355e6c, // foam and hollow water
		'f': 0xe87432, '*': 0xffd778, // fire and incandescent core
		's': 0x53616b, '#': 0xa1adb0, // submerged wreck and broken metal
	}
	// A slow traveling luminance wave keeps the silhouette fixed and legible.
	// Cell offsets prevent the whole chart pulsing in synchrony.
	boil := func(c RGB, px, py int) RGB {
		if mark == game.Sunk {
			return c
		}
		wave := math.Sin(t*1.5 + float64(x+y)*.71 + float64(px)*.8 - float64(py)*.65)
		if wave > 0 {
			return mix(c, 0xffedbb, wave*.09)
		}
		return mix(c, 0x173841, -wave*.13)
	}
	var sprite []string
	switch mark {
	case game.Miss:
		sprite = []string{"..~~~..", ".~ooo~.", "~ooooo~", ".~ooo~.", "..~~~.."}
	case game.Hit:
		sprite = []string{"...f...", ".f.*...", ".ff*f..", "ff***f.", ".fffff."}
	case game.Sunk:
		sprite = []string{"....#..", "..s#s..", ".s###s.", "s##s##s", ".sssss."}
	default:
		return
	}
	if h < 4 || w < 5 {
		switch mark {
		case game.Miss:
			sprite = []string{"~o~", ".~."}
		case game.Hit:
			sprite = []string{".*.", "fff"}
		case game.Sunk:
			sprite = []string{".#.", "#s#"}
		}
	}
	if w == 1 || h == 1 {
		color := palette['~']
		if mark == game.Hit {
			color = palette['*']
		} else if mark == game.Sunk {
			color = palette['#']
		}
		r.rect(x, y, w, h, boil(color, 0, 0))
		return
	}
	for py := 0; py < h; py++ {
		row := sprite[py*len(sprite)/h]
		for px := 0; px < w; px++ {
			if color, ok := palette[rune(row[px*len(row)/w])]; ok {
				r.put(x+px, y+py, boil(color, px, py))
			}
		}
	}
}
