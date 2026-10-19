package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sea-battle-2/internal/bot"
	"sea-battle-2/internal/game"
)

func main() {
	for _, pair := range [][2]bot.Difficulty{{bot.Medium, bot.Medium}, {bot.Hard, bot.Medium}, {bot.Medium, bot.Easy}} {
		counts := map[int]int{}
		n := 48
		for seed := 0; seed < n; seed++ {
			g := game.New(seed % 2)
			g.SetFleet(0, game.RandomFleet(uint64(14000+seed*2)))
			g.SetFleet(1, game.RandomFleet(uint64(14001+seed*2)))
			rng := rand.New(rand.NewPCG(uint64(seed+91), 56))
			for g.Phase == "battle" {
				p := g.Turn
				at, ok := bot.Decide(context.Background(), pair[p], bot.Observe(g.View(p)), rng)
				if !ok {
					panic("bot could not choose a legal shot")
				}
				if err := g.Fire(p, at, len(g.Events)); err != nil {
					panic(err)
				}
			}
			hp := game.WinnerHullRemaining(g.View(g.Winner))
			for _, cut := range []int{8, 9, 10, 11, 12} {
				if hp >= cut {
					counts[cut]++
				}
			}
		}
		fmt.Println(pair, n, counts)
	}
}
