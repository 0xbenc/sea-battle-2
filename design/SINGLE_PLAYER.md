# Single player

Pick **S — Single player**, then **A — American navy** or **B — The brood**.
The computer commands the opposite faction. Both use the same classic rules,
five hull lengths (2, 3, 3, 4, 5), placement controls, ocean, consoles, hit/sink
cinematics and victory/defeat presentation as multiplayer. Human goes first in
match one; rematches alternate the opener and preserve faction choice. R at the
result starts another deployment; ESC returns to the menu (confirmation during
an unfinished match). No difficulty submenu in the first version.

This is fully local: no listening socket, TLS connection, invitation, router
mapping or shared service. `--solo american` and `--solo alien` bypass the menus.
The session controller owns the local game, bot and cancellable turn timer, with
the same generation-tagged events and command validation used by presentation.
Multiplayer remains a separate backend of that controller. Player index denotes
faction, not whether the player is human or host.

## A competent, fair opponent

The bot receives only its outgoing shot marks and identities of ships it has
sunk. It cannot access either fleet, incoming shots, the authoritative game, or
placement randomness. Separate random streams govern deployment and targeting.
For each remaining hull (including both length-three hulls), enumerate all
straight placements that avoid misses and sunk cells. Score unknown squares by
how many candidates cover them. Prioritize candidates containing unresolved
hits, weighting those covering several hits more strongly, and randomly break
score ties. This is a placement-density heuristic, not an exact probability or
an omniscient solver. Adjacent ships are legal: do not assume a water border or
that connected hits all belong to one ship. Sinking removes precisely the
revealed hull; other hits remain targets. If scoring yields no target, choose
an untried cell, never repeat a shot.

The computer locks a random legal fleet before human deployment. It submits one
shot after a short 650 ms delay whenever it owns the turn. Thinking is independent
of frame rate, focus and terminal size. Results queue into the existing skippable
cinematics; a response can resolve while the preceding shot is being shown, but
its transmission plays in order. No second bot shot can happen until the human
fires again. Leaving, restarting or quitting cancels and joins the old session;
queued commands cannot affect a new match or session. No save/resume across app
exits in this version.

## Acceptance checks

Exercise both factions, all legal shot/placement rules, rematch opener alternation,
repeated inputs, cancellation during thinking and stale queued commands. Run
large seeded bot matches including touching ships and duplicate hull lengths;
compare against random shooting and verify targeting depends only on public
knowledge. Test real terminal menu/CLI entry, deployment, bot response, cutscene
skip, return to menu and terminal restoration at SD and HD. Retain multiplayer,
lookdev, race detector and vet regression checks. Record measured results below
once implemented; do not infer real network coverage from local tests.

## Implemented and verified

- 500 seeded games against random shooting: 500 bot wins, 44.64 mean bot shots;
  alternating opening seats. This measures a baseline, not human difficulty.
- 24 complete controller-driven games (12 per human faction), including repeated
  command rejection, fresh rematch fleets and alternating first shots.
- Targeting checks cover touching ships, exhausted targets, duplicate length-three
  hull identities and exclusion of private/defender information.
- Session cancellation interrupts the think timer; prepared commands cannot cross
  generations. Presentation checks verify both factions and ordered human/bot
  cutscenes even when the bot responds before the first transmission finishes.
- 42 real-terminal checks cover both factions at SD and HD, menu/CLI entry,
  deployment, a bot response, cutscene skip, leave/cancel/menu and terminal cleanup.
  Captures were visually reviewed. The multiplayer terminal suite also passes.
- `go test -race ./...` and `go vet ./...` pass. `make test-terminal` includes
  the new solo checks alongside multiplayer and lookdev regression suites.

The first implementation iteration also fixed a delayed stop event dismissing
an already-opened faction picker. Gameplay remains in memory; exiting abandons
that local match. There is one bot strength, with no adaptive cheating.
