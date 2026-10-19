package game

// OverwhelmingHP requires the victor to preserve more than half of the fleet.
const OverwhelmingHP = 9

// WinnerHullRemaining uses confirmed damage, so both filtered views agree
// without revealing any surviving enemy placement or adding protocol fields.
func WinnerHullRemaining(v View) int {
	if v.Phase != "finished" || v.Winner < 0 || v.Winner > 1 {
		return 0
	}
	marks := v.Incoming
	if v.Winner != v.Player {
		marks = v.Outgoing
	}
	hp := 17
	for _, mark := range marks {
		if mark == Hit || mark == Sunk {
			hp--
		}
	}
	return max(0, hp)
}
