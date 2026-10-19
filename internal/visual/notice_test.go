package visual

import (
	"testing"
	"unicode/utf8"
)

func TestNoticeReview(t *testing.T) {
	for _, sd := range []bool{true, false} {
		capture(t, RenderProblem(sd, 1, MenuIdle{}, "Another game is using this port. Close it, then try hosting again.", ""), "problem", sd, 0)
		capture(t, RenderJoin(sd, 1, MenuIdle{}, "That invitation looks incomplete. Please paste the whole thing.", "", false), "join-error", sd, 0)
		for team := 0; team < 2; team++ {
			b := combatFixture(team, sd)
			b.Solo = false
			b.Host = team == 0
			b.State.Connected[1] = false
			capture(t, RenderBattle(b), "recovery", sd, team)
		}
	}
}

func TestFeedbackTruncationPreservesCharacters(t *testing.T) {
	for width := 0; width < 20; width++ {
		s := short("Connecting… café résumé", width)
		if !utf8.ValidString(s) || utf8.RuneCountInString(s) > width {
			t.Fatalf("invalid width %d: %q", width, s)
		}
	}
}
