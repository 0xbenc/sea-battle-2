package game

import "testing"

func TestWinnerHullRemaining(t *testing.T) {
	for winner := 0; winner < 2; winner++ {
		for _, hp := range []int{1, 8, 9, 17} {
			for viewer := 0; viewer < 2; viewer++ {
				v := View{Phase: "finished", Winner: winner, Player: viewer}
				marks := &v.Incoming
				if viewer != winner {
					marks = &v.Outgoing
				}
				for i := 0; i < 17-hp; i++ {
					marks[i] = Hit
					if i%2 == 0 {
						marks[i] = Sunk
					}
				}
				marks[99] = Miss
				if got := WinnerHullRemaining(v); got != hp {
					t.Fatal(winner, viewer, hp, got)
				}
				if (WinnerHullRemaining(v) >= OverwhelmingHP) != (hp >= 9) {
					t.Fatal("threshold")
				}
			}
		}
	}
	if WinnerHullRemaining(View{Phase: "battle"}) != 0 {
		t.Fatal("unfinished game has a result")
	}
}
