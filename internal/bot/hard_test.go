package bot

import (
	"context"
	"sea-battle-2/internal/game"
	"testing"
	"time"
)

func cells(ns ...int) mask {
	m := mask{}
	for _, n := range ns {
		m = m.or(bit(n))
	}
	return m
}
func TestExactJointFleetInference(t *testing.T) {
	p := posterior{candidates: [][]mask{{cells(0, 1), cells(1, 2)}, {cells(2, 3), cells(3, 4)}}, hits: cells(1), unknown: cells(0, 2, 3, 4)}
	hunt, target, ok := p.scores(context.Background(), random(10))
	if !ok {
		t.Fatal("no hypotheses")
	}
	want := map[int]int{0: 2, 2: 2, 3: 3, 4: 2}
	for i, n := range hunt {
		if n != want[i] {
			t.Fatal("joint overlap accounting", i, n, want[i])
		}
	}
	if target[0] != 120 || target[2] != 60 || target[3] != 0 || target[4] != 0 {
		t.Fatal("hit ownership", target)
	}
}
func TestSamplerConditionalFrequenciesAndHitCoverage(t *testing.T) {
	p := posterior{candidates: [][]mask{{cells(0, 1), cells(2, 3)}, {cells(10, 11), cells(12, 13)}, {cells(20, 21), cells(22, 23)}}, hits: cells(0), unknown: cells(1, 2, 3, 10, 11, 12, 13, 20, 21, 22, 23)}
	hunt, target, ok := p.scores(context.Background(), random(10))
	if !ok || hunt[1] != 1536 || hunt[2] != 0 || target[1] != 1536*60 {
		t.Fatal("sampler omitted a known hit")
	}
	for _, i := range []int{10, 11, 12, 13, 20, 21, 22, 23} {
		if hunt[i] < 600 || hunt[i] > 940 {
			t.Fatal("independent equal hypotheses not mixed", i, hunt[i])
		}
	}
}
func TestOptimalSmallEndgame(t *testing.T) {
	// Shooting shared cell 1 costs 8/3 expected shots; either isolated group
	// costs at least that much. The overlapping group must be investigated first.
	p := posterior{candidates: [][]mask{{cells(0, 1), cells(1, 2), cells(3, 4)}}, unknown: cells(0, 1, 2, 3, 4)}
	at, ok := optimalEndgame(context.Background(), p, random(2))
	if !ok || at.Index() != 1 {
		t.Fatal("wrong decision tree", at)
	}
	p.hits = cells(0)
	p.unknown = p.unknown.without(p.hits)
	at, ok = optimalEndgame(context.Background(), p, random(2))
	if !ok || at.Index() != 1 {
		t.Fatal("inconsistent hypotheses retained", at)
	}
}
func TestAllDifficultiesRememberHitsAndAllowTouching(t *testing.T) {
	for _, d := range []Difficulty{Easy, Medium, Hard} {
		k := Knowledge{Sunk: [5]bool{true, false, true, true, true}}
		for i := range k.Shots {
			k.Shots[i] = game.Miss
		}
		k.Shots[0], k.Shots[1] = game.Sunk, game.Sunk
		k.Shots[10], k.Shots[11], k.Shots[12] = game.Hit, game.Hit, game.Unknown
		for seed := uint64(0); seed < 20; seed++ {
			at, ok := Decide(context.Background(), d, k, random(seed))
			if !ok || at.Index() != 12 {
				t.Fatal(d, "forgot live hits beside sunk hull", at)
			}
		}
	}
}
func TestHardCancellationDuringInference(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); possibilities(Knowledge{}).scores(ctx, random(1)) }()
	time.Sleep(time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("inference ignored cancellation")
	}
}
