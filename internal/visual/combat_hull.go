package visual

import "sea-battle-2/internal/game"

// The SD overview has one interior half-pixel per cell: reducing detailed art
// here erases bows and breaks hulls apart. Draw a continuous tactical silhouette.
func (r *Raster) combatSmallHull(b BattlePresentation, g CombatBoard, index int, p game.Placement) {
	hull, spine := RGB(0xbacbd1), RGB(0x668d9b)
	if b.State.Player == 1 {
		hull, spine = 0xb5dca2, 0x836b99
	}
	x, y := g.X+p.X*g.CW+1, g.Y+p.Y*g.CH+1
	w, h := g.CW-1, game.Lengths[index]*g.CH-1
	if p.Horizontal {
		w, h = game.Lengths[index]*g.CW-1, g.CH-1
	}
	// Dark water edging separates adjacent hulls and brightens their silhouette.
	r.rect(x-1, y-1, w+2, h+2, 0x0b202b)
	r.rect(x, y, w, h, spine)
	for _, c := range game.Cells(p, game.Lengths[index]) {
		r.rect(g.X+c.X*g.CW+1, g.Y+c.Y*g.CH+1, g.CW-1, g.CH-1, hull)
	}
}
