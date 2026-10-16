// Package game is the authoritative, transport-independent classic Battleship engine.
package game

import (
	"errors"
	"fmt"
	"math/rand/v2"
)

const Size = 10

var Lengths = [5]int{2, 3, 3, 4, 5}

type Coord struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func (c Coord) Valid() bool    { return c.X >= 0 && c.X < Size && c.Y >= 0 && c.Y < Size }
func (c Coord) Index() int     { return c.Y*Size + c.X }
func (c Coord) String() string { return fmt.Sprintf("%c%d", 'A'+c.X, c.Y+1) }

type Placement struct {
	Coord
	Horizontal bool `json:"horizontal"`
}

func Cells(p Placement, length int) []Coord {
	out := make([]Coord, length)
	for i := range out {
		out[i] = p.Coord
		if p.Horizontal {
			out[i].X += i
		} else {
			out[i].Y += i
		}
	}
	return out
}
func CanPlace(fleet []Placement, index int, p Placement) bool {
	if index < 0 || index >= len(Lengths) {
		return false
	}
	occupied := [100]bool{}
	for i, s := range fleet {
		if i >= len(Lengths) || i == index || !s.Valid() {
			continue
		}
		for _, c := range Cells(s, Lengths[i]) {
			if c.Valid() {
				occupied[c.Index()] = true
			}
		}
	}
	for _, c := range Cells(p, Lengths[index]) {
		if !c.Valid() || occupied[c.Index()] {
			return false
		}
	}
	return true
}
func ValidateFleet(f []Placement) error {
	if len(f) != 5 {
		return errors.New("place all five ships")
	}
	for i, p := range f {
		if !CanPlace(f, i, p) {
			return fmt.Errorf("ship %d overlaps another hull or crosses the board edge", i+1)
		}
	}
	return nil
}
func EmptyFleet() []Placement {
	f := make([]Placement, 5)
	for i := range f {
		f[i].X = -1
		f[i].Y = -1
	}
	return f
}
func RandomFleet(seed uint64) []Placement {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	f := EmptyFleet()
	for i := 4; i >= 0; i-- {
		candidates := rng.Perm(200)
		for _, n := range candidates {
			p := Placement{Coord{n % 10, (n / 10) % 10}, n >= 100}
			if CanPlace(f, i, p) {
				f[i] = p
				break
			}
		}
	}
	return f
}

type Mark uint8

const (
	Unknown Mark = iota
	Miss
	Hit
	Sunk
)

type Event struct {
	Seq    int   `json:"seq"`
	Player int   `json:"player"`
	At     Coord `json:"at"`
	Hit    bool  `json:"hit"`
	Sunk   bool  `json:"sunk"`
	Ship   int   `json:"ship"`
}
type View struct {
	Player    int         `json:"player"`
	Phase     string      `json:"phase"`
	Turn      int         `json:"turn"`
	Winner    int         `json:"winner"`
	Ready     [2]bool     `json:"ready"`
	Connected [2]bool     `json:"connected"`
	Own       []Placement `json:"own,omitempty"`
	Incoming  [100]Mark   `json:"incoming"`
	Outgoing  [100]Mark   `json:"outgoing"`
	Events    []Event     `json:"events,omitempty"`
	Rematch   [2]bool     `json:"rematch"`
	Match     int         `json:"match"`
}
type Game struct {
	fleet               [2][]Placement
	shots               [2][100]Mark // shooter -> target grid
	Ready               [2]bool
	Turn, Winner, Match int
	first               int
	Phase               string
	Events              []Event
	Rematch             [2]bool
}

func New(first int) *Game {
	return &Game{first: first % 2, Turn: first % 2, Winner: -1, Phase: "placement", Match: 1}
}
func (g *Game) SetFleet(player int, f []Placement) error {
	if player < 0 || player > 1 {
		return errors.New("invalid player")
	}
	if g.Phase != "placement" {
		return errors.New("battle already started")
	}
	if g.Ready[player] {
		return errors.New("fleet already locked")
	}
	if err := ValidateFleet(f); err != nil {
		return err
	}
	g.fleet[player] = append([]Placement(nil), f...)
	g.Ready[player] = true
	if g.Ready[0] && g.Ready[1] {
		g.Phase = "battle"
	}
	return nil
}
func (g *Game) Fire(player int, at Coord, expected int) error {
	if player < 0 || player > 1 {
		return errors.New("invalid player")
	}
	if g.Phase != "battle" {
		return errors.New("battle is not active")
	}
	if player != g.Turn {
		return errors.New("wait for your turn")
	}
	if expected != len(g.Events) {
		return errors.New("battle changed; choose your shot again")
	}
	if !at.Valid() {
		return errors.New("target is outside A1-J10")
	}
	if g.shots[player][at.Index()] != Unknown {
		return errors.New("that square has already been fired on")
	}
	defender := 1 - player
	ship := -1
	for i, p := range g.fleet[defender] {
		for _, c := range Cells(p, Lengths[i]) {
			if c == at {
				ship = i
			}
		}
	}
	e := Event{Seq: len(g.Events) + 1, Player: player, At: at, Ship: -1}
	if ship < 0 {
		g.shots[player][at.Index()] = Miss
	} else {
		g.shots[player][at.Index()] = Hit
		e.Hit = true
		all := true
		for _, c := range Cells(g.fleet[defender][ship], Lengths[ship]) {
			if g.shots[player][c.Index()] < Hit {
				all = false
			}
		}
		if all {
			e.Sunk = true
			e.Ship = ship
			for _, c := range Cells(g.fleet[defender][ship], Lengths[ship]) {
				g.shots[player][c.Index()] = Sunk
			}
		}
	}
	g.Events = append(g.Events, e)
	g.Turn = defender
	hits := 0
	for _, m := range g.shots[player] {
		if m >= Hit {
			hits++
		}
	}
	if hits == 17 {
		g.Phase = "finished"
		g.Winner = player
	}
	return nil
}
func (g *Game) RequestRematch(player int) error {
	if player < 0 || player > 1 || g.Phase != "finished" {
		return errors.New("finish the battle before requesting a rematch")
	}
	g.Rematch[player] = true
	if g.Rematch[0] && g.Rematch[1] {
		match := g.Match + 1
		*g = *New(1 - g.first)
		g.Match = match
	}
	return nil
}

// View exposes no enemy placement or unsunk enemy ship identity, even after defeat.
func (g *Game) View(player int) View {
	v := View{Player: player, Phase: g.Phase, Turn: g.Turn, Winner: g.Winner, Ready: g.Ready, Own: append([]Placement(nil), g.fleet[player]...), Incoming: g.shots[1-player], Outgoing: g.shots[player], Events: append([]Event(nil), g.Events...), Rematch: g.Rematch, Match: g.Match}
	// A defender already knows their own hull: allow its actual cinematic identity.
	for i := range v.Events {
		e := &v.Events[i]
		if e.Player != player && e.Hit && e.Ship < 0 {
			for id, p := range g.fleet[player] {
				for _, c := range Cells(p, Lengths[id]) {
					if c == e.At {
						e.Ship = id
					}
				}
			}
		}
	}
	return v
}
