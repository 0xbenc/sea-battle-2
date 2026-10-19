package play

import "strings"

func actionError(message string) string {
	text := strings.ToLower(message)
	switch {
	case strings.Contains(text, "address already in use"), strings.Contains(text, "only one usage"):
		return "Another game is using this port. Close it, then try hosting again."
	case strings.Contains(text, "expired"):
		return "That invitation has expired. Ask your friend for a new one."
	case strings.Contains(text, "rejected"), strings.Contains(text, "fingerprint"), strings.Contains(text, "certificate"):
		return "That invitation didn't match the host. Ask for a fresh copy."
	case strings.Contains(text, "already been fired"):
		return "You've already tried that square. Pick another one."
	case strings.Contains(text, "wait for your turn"):
		return "It's your opponent's turn."
	case strings.Contains(text, "place all five"):
		return "Place all five ships before you launch."
	case strings.Contains(text, "overlaps"), strings.Contains(text, "board edge"):
		return "Ships need to fit on the grid without overlapping."
	case strings.Contains(text, "fleet already locked"):
		return "Your fleet is ready. Waiting for your opponent."
	case strings.Contains(text, "battle changed"):
		return "The board changed. Pick your shot again."
	case strings.Contains(text, "connection"), strings.Contains(text, "closed"), strings.Contains(text, "timeout"):
		return "Connection interrupted. Please try connecting again."
	default:
		return "That didn't work. Please try again."
	}
}
