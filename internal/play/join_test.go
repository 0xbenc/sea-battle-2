package play

import (
	tea "charm.land/bubbletea/v2"
	"encoding/base64"
	"sea-battle-2/internal/peer"
	"strings"
	"testing"
	"time"
)

func joinTestCode(t *testing.T) string {
	t.Helper()
	code, err := (peer.Invitation{Version: peer.Version, Addresses: []string{"127.0.0.1:1"}, Fingerprint: strings.Repeat("a", 64), Token: base64.RawURLEncoding.EncodeToString(make([]byte, 32)), Expires: time.Now().Add(time.Hour).Unix()}).Encode()
	if err != nil {
		t.Fatal(err)
	}
	return code
}
func TestAutomaticJoinInputs(t *testing.T) {
	code := joinTestCode(t)
	for _, source := range []string{"paste", "terminal", "native", "typed"} {
		t.Run(source, func(t *testing.T) {
			m, close := New(Options{})
			defer close()
			m.width, m.height = 80, 24
			m = apply(m, key("j"))
			switch source {
			case "paste":
				m = apply(m, tea.PasteMsg{Content: code[:20] + "\n" + code[20:]})
			case "terminal":
				m = apply(m, tea.ClipboardMsg{Selection: 'c', Content: code})
			case "native":
				m = apply(m, clipboardRead{serial: m.clipboardSerial, content: code})
			case "typed":
				for _, r := range code {
					m = apply(m, tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
				}
			}
			if m.mode != "connecting" || m.input != code || m.clipboardPending {
				t.Fatalf("%s did not auto-connect: %s", source, m.mode)
			}
		})
	}
}
func TestJoinIgnoresUnrelatedAndLateClipboard(t *testing.T) {
	m, close := New(Options{})
	defer close()
	m.width, m.height = 80, 24
	m = apply(m, key("j"))
	m = apply(m, clipboardRead{serial: m.clipboardSerial, content: "unrelated private clipboard text"})
	if m.mode != "join" || m.input != "" {
		t.Fatal("unrelated clipboard used")
	}
	m = apply(m, tea.PasteMsg{Content: "sb2:incomplete"})
	if m.mode != "join" || m.clipboardPending || m.input != "" {
		t.Fatal("invalid paste started connection")
	}
	m = apply(m, tea.ClipboardMsg{Selection: 'c', Content: joinTestCode(t)})
	if m.mode != "join" {
		t.Fatal("late clipboard superseded manual paste")
	}
	m = apply(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	m = apply(m, clipboardRead{serial: m.clipboardSerial, content: joinTestCode(t)})
	if m.mode != "menu" {
		t.Fatal("clipboard re-opened cancelled join")
	}
}
