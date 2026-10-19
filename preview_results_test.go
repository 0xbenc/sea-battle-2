package main

import (
	"sea-battle-2/internal/game"
	"testing"
)

func TestResultPreviewCoversAllOutcomes(t *testing.T) {
	for i := 0; i < 8; i++ {
		v := resultPreviewState(i)
		if (v.Player == v.Winner) != (i%4 < 2) {
			t.Fatal("wrong outcome", i)
		}
		hp := 4
		if i%2 == 1 {
			hp = 12
		}
		if game.WinnerHullRemaining(v) != hp {
			t.Fatal("wrong hull remaining", i)
		}
		if game.ValidateFleet(v.Own) != nil {
			t.Fatal("invalid preview fleet", i)
		}
	}
}
