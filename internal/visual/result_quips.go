package visual

import (
	_ "embed"
	"encoding/json"
)

// Edit result_quips.json, then rebuild with make preview-results or make run.
//
//go:embed result_quips.json
var resultQuipsJSON []byte

var resultQuips = func() map[string]map[string]string {
	var quips map[string]map[string]string
	if err := json.Unmarshal(resultQuipsJSON, &quips); err != nil {
		panic("invalid result_quips.json: " + err.Error())
	}
	return quips
}()

func resultQuip(player int, won, dominant bool) string {
	faction := "united_states_navy"
	if player == 1 {
		faction = "the_brood"
	}
	outcome := "close_victory"
	if won && dominant {
		outcome = "overwhelming_victory"
	}
	if !won {
		outcome = "narrow_defeat"
		if dominant {
			outcome = "overwhelming_defeat"
		}
	}
	return resultQuips[faction][outcome]
}
