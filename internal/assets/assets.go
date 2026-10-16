// Package assets validates and indexes the embedded Dotmason v1 artwork once.
package assets

import (
	"bytes"
	_ "embed"
	"fmt"
	"git.occ.farm/occ/dotmason/art"
)

//go:embed fleet.mason
var source []byte
var Doc *art.Document
var canvases map[string]*art.Canvas
var Palette []uint32
var Names = []string{"Cyclone", "Virginia", "Freedom", "Zumwalt", "Gerald R. Ford", "Drone Skiff", "Deep Lurker", "Littoral Ravager", "Siege Carapace", "Hive Carrier"}
var Classes = []string{"PATROL BOAT", "ATTACK SUBMARINE", "LITTORAL COMBAT SHIP", "DESTROYER", "AIRCRAFT CARRIER", "DRONE SKIFF", "DEEP LURKER", "LITTORAL RAVAGER", "SIEGE CARAPACE", "HIVE CARRIER"}
var Lengths = []int{2, 3, 3, 4, 5, 2, 3, 3, 4, 5}

func init() {
	var err error
	Doc, err = art.Decode(bytes.NewReader(source))
	if err != nil {
		panic(err)
	}
	canvases = map[string]*art.Canvas{}
	Palette = make([]uint32, len(Doc.Palette)+1)
	for i, c := range Doc.Palette {
		var n uint32
		if _, err = fmt.Sscanf(c, "#%06x", &n); err != nil {
			panic(err)
		}
		Palette[i+1] = n
	}
	for i := range Doc.Designs {
		d := &Doc.Designs[i]
		for j := range d.LODs {
			c := &d.LODs[j]
			canvases[d.ID+"/"+c.Name] = c
		}
	}
	for i := 0; i < 10; i++ {
		for _, view := range []string{"side", "top", "front"} {
			for _, size := range []string{"HD", "SD"} {
				Get(fmt.Sprintf("ship-%d-%s", i, view), size)
			}
		}
		for _, size := range []string{"board-HD", "board-SD", "board-horizontal-HD", "board-horizontal-SD"} {
			Get(fmt.Sprintf("ship-%d-top", i), size)
		}
	}
	for _, size := range []string{"HD", "SD"} {
		c := Get("impact", size)
		if _, err := c.Track("burst"); err != nil {
			panic(err)
		}
	}
}
func Get(id, size string) *art.Canvas {
	c, ok := canvases[id+"/"+size]
	if !ok {
		panic("missing art: " + id + "/" + size)
	}
	return c
}
