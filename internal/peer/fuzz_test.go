package peer

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"sea-battle-2/internal/game"
)

func FuzzInvitation(f *testing.F) {
	i := Invitation{Version: Version, Addresses: []string{"127.0.0.1:4747", "[2001:db8::1]:4747"}, Fingerprint: hex.EncodeToString(make([]byte, 32)), Token: base64.RawURLEncoding.EncodeToString(make([]byte, 32)), Expires: time.Now().Add(time.Hour).Unix()}
	code, _ := i.Encode()
	f.Add(code)
	f.Add("sb2:")
	f.Add("")
	f.Add("sb2:\x1b[31m")
	f.Fuzz(func(t *testing.T, s string) {
		i, err := ParseInvitation(s)
		if err != nil {
			return
		}
		code, err := i.Encode()
		if err != nil {
			t.Fatal(err)
		}
		copy, err := ParseInvitation(code)
		if err != nil || !reflect.DeepEqual(i, copy) {
			t.Fatal("invitation roundtrip")
		}
	})
}
func FuzzWireMessages(f *testing.F) {
	g := game.New(0)
	v := g.View(0)
	b, _ := json.Marshal(Message{Type: "state", View: &v})
	f.Add(b)
	g.SetFleet(0, game.RandomFleet(1))
	enemy := game.RandomFleet(2)
	g.SetFleet(1, enemy)
	g.Fire(0, game.Cells(enemy[0], game.Lengths[0])[0], 0)
	for player := 0; player < 2; player++ {
		battle := g.View(player)
		encoded, _ := json.Marshal(Message{Type: "state", View: &battle})
		f.Add(encoded)
	}
	f.Add([]byte(`{"type":"fatal","error":"\u001b[31m"}`))
	f.Add([]byte(`{"type":"state","view":{"player":-1}}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		m, err := decodeMessage(data)
		if err != nil {
			return
		}
		if len(m.Error) > 240 {
			t.Fatal("unbounded display message")
		}
		for _, r := range m.Error {
			if r < 32 || r > 126 {
				t.Fatal("terminal control escaped decoder")
			}
		}
		if m.Type == "state" && !validView(m.View) {
			t.Fatal("invalid state escaped decoder")
		}
	})
}
func FuzzCommandsPreserveRules(f *testing.F) {
	f.Add([]byte(`{"type":"fire","at":{"x":0,"y":0},"expected":0,"match":1}`))
	f.Add([]byte(`{"type":"ready","fleet":[]}`))
	fleet, _ := json.Marshal(Command{Type: "ready", Fleet: game.RandomFleet(42)})
	f.Add(fleet)
	f.Fuzz(func(t *testing.T, data []byte) {
		cmd, err := decodeCommand(data)
		if err != nil {
			return
		}
		g := game.New(0)
		if cmd.Type != "ready" {
			g.SetFleet(0, game.RandomFleet(1))
			g.SetFleet(1, game.RandomFleet(2))
		}
		before := g.View(0)
		switch cmd.Type {
		case "fire":
			err = g.Fire(0, cmd.At, cmd.Expected)
		case "ready":
			err = g.SetFleet(0, cmd.Fleet)
		case "rematch":
			err = g.RequestRematch(0)
		default:
			return
		}
		after := g.View(0)
		if err != nil && !reflect.DeepEqual(before, after) {
			t.Fatal("rejection mutated match")
		}
		if !validView(&after) {
			t.Fatal("command produced invalid state")
		}
		if len(after.Events) > 1 {
			t.Fatal("one command made multiple shots")
		}
	})
}
