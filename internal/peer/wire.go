package peer

import (
	"encoding/json"
	"errors"
	"strings"
)

// All network decoders share the same byte bound as the line scanners.
func decodeMessage(data []byte) (Message, error) {
	var m Message
	if len(data) > maxMessage || json.Unmarshal(data, &m) != nil {
		return m, errors.New("host sent invalid data")
	}
	switch m.Type {
	case "state":
		if !validView(m.View) || (m.Transport != "" && m.Transport != "direct" && m.Transport != "public") {
			return Message{}, errors.New("host sent invalid state")
		}
	case "invitation":
		if m.Invitation == nil || m.Invitation.Validate() != nil {
			return Message{}, errors.New("host sent invalid invitation")
		}
	case "error", "fatal", "pong", "probe_ok":
	default:
		return Message{}, errors.New("host sent an unknown message type")
	}
	m.Error = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 {
			return ' '
		}
		return r
	}, m.Error)
	if len(m.Error) > 240 {
		m.Error = m.Error[:240]
	}
	return m, nil
}
func decodeCommand(data []byte) (Command, error) {
	var c Command
	if len(data) > maxMessage || json.Unmarshal(data, &c) != nil {
		return c, errors.New("invalid command")
	}
	switch c.Type {
	case "hello", "ping", "ready", "fire", "rematch", "probe", "link_ready":
		return c, nil
	default:
		return Command{}, errors.New("unknown command")
	}
}
