# Single player

Press **S**, select **1 Easy / 2 Medium / 3 Hard**, then **A United States Navy** or
**B The brood**. Medium is the default. The computer commands the opposite faction.
`--solo american --difficulty hard` (or `--solo alien`) bypasses the picker.

All difficulties use the same classic rules and random legal fleet deployment:
five hull lengths (2, 3, 3, 4, 5), touching allowed, one shot per turn even on a hit.
Human opens match one; rematches alternate the opener and preserve faction and
difficulty. R after the result starts deployment again. ESC returns to the menu,
with confirmation before abandoning an unfinished match. Matches live in memory.

This is fully local: no listening socket, TLS, invitation, router mapping or
shared service. The session controller owns the game, bot and cancellable turn
timer. It uses the same generation-tagged event/command boundary as multiplayer.
Player index means faction, not human/host. Quitting joins all session work,
including inference; stale commands cannot reach another session or match.

## Three fair opponents

Every bot receives only its own outgoing shot marks and identities of ships it
has sunk. No enemy fleet, incoming shots, authoritative game or placement seed is
available to targeting. Difficulty never changes the rules, reveals a square,
repositions a ship or manipulates an outcome. Deployment and targeting use
independent randomness. Two length-three hulls remain separate identities.

- **Easy:** search randomly among untried squares, then follow unresolved hits
  through viable neighboring cells until they sink. It never deliberately forgets
  a live hit. It does not optimize orientation or global search coverage.
- **Medium:** search randomly among plausible squares, excluding impossible cells. Follow hits
  using legal straight-hull candidates, strongly favoring aligned multiple hits.
  This is a conventional hunt-and-target opponent, without joint fleet inference.
- **Hard:** when following hits, infer complete nonoverlapping remaining fleets
  consistent with every miss, sunk cell and unresolved hit. With three or more
  ships, use four randomized
  Gibbs chains, each with 256 burn-in steps and 768 further steps, observing every
  other step: 1,536 complete-fleet samples. Single- and two-hull conditional updates
  let touching ships exchange possible ownership of hits. Initialization uses
  bounded, pruned backtracking. With one or two ships, enumerate legal joint fleets
  exactly. Weight hit-bearing hulls by how few additional hits they need to sink.
  When hunting with more than two hulls left, use the complete surface of legal
  per-hull placements directly; with one or two, use joint inference and search
  for the smallest hull. This hybrid outperformed using sampled marginals for
  every early hunt shot. No parity restriction is imposed on Hard.

After at least two sunk hulls reveal a dense cluster (at least half of a bounding
rectangle no wider or taller than six cells), Hard increases search weight within
two cells of that cluster. This is an explicit placement-pattern heuristic, not a
uniform-prior probability. It addresses tightly packed fleets without treating
every isolated sunk ship as evidence that the rest must be nearby.

For one remaining hull and at most 18 consistent placements, Hard also solves
an adaptive hit/miss decision tree minimizing expected additional shots. Factor
out guaranteed hits, memoize subproblems and cap search at 40,000 recursive calls.
If that search exceeds its budget, use the inferred shot scores. If joint fleet
initialization cannot complete within its budget, fall back to the Medium policy.
All inference paths support cancellation; none is allowed to stall shutdown.

**Mathematical scope:** exact enumeration assumes a uniform prior over legal,
labeled remaining fleets. Actual human placement habits and the random deployment
generator need not follow that prior. Larger-fleet inference and the hunting policy
are approximations; finite sampling does not prove convergence. The small endgame
solver is exact only when its search completes. Hard is intended to be formidable,
but neither global optimality nor a requirement for luck to beat it is established.
Do not label it a perfect solver based on a bot tournament alone.

## Presentation advances with the shot

The model keeps the newest authoritative state for command validation separately
from the visible state. Every live shot queues **one immutable item containing its
event and post-shot board snapshot**. While its transmission plays, the visible
board is still the preceding snapshot. Completion or skip commits exactly that
shot's result. Later damage, turn changes, last-shot text and victory/defeat remain
hidden until their own transmission completes. This applies to both game modes.

Non-shot updates can update an idle presentation. Connection health is immediate,
even during playback. Initial connection, reconnect, a changed match or a multi-shot
catch-up replaces the visible snapshot and drops queued history; it does not invent
intermediate boards from a final snapshot. Controls remain gated during playback,
and command sequence checks always use the newest authority.

The bot begins computing after a 650 ms turn delay, independent of terminal focus
or frame rate. Its result may arrive during the previous transmission, but playback
stays ordered. Misses last 2 seconds, hits 3.5, sinks 6. A bot transmission begins
with an additional 0.8-second enemy-turn beat that repeated skip keys cannot erase.
Clips identify outgoing versus incoming fire. There cannot be another bot shot
until the human fires again. Charts use centered O/X/# marks, stable shaded cells,
visible grid lines, interior cursor brackets and highlighted coordinate headers.

## Reproducible checks

`go test -race ./...`, `go vet ./...`, and `make test-terminal` cover:

- 24 full controller-driven games: four matches per faction per difficulty,
  including repeated/stale commands, opener alternation and fresh fleets.
- Bot legality, touching hulls, live-hit follow-up at every difficulty, distinct
  length-three identities, exact joint counts, sampler conditional frequencies,
  small optimal endgames and cancellation during inference.
- Snapshot ownership, caller mutation isolation, ordered damage/turn/result
  visibility, skip, catch-up and reconnect while playback is queued.
- 54 real-terminal checks for solo faction/difficulty selection, SD/HD play,
  handoff, bot response, leave/cancel and cleanup; CLI entry for all six
  faction/difficulty combinations. Multiplayer and lookdev terminal regressions.

The ordinary suite compares difficulties on 24 common hidden boards and eight
corner-packed, touching fleets. Expand the independent holdout evaluation with:

```sh
SB2_BOT_EVAL_GAMES=160 SB2_BOT_EVAL_SEED=400000 \
  go test ./internal/bot -run 'TestDifficultyEvaluation|TestDifficultiesOnTouching' -v
```

It measures shots to clear each board, then pairs different boards into races,
alternating first player (which resolves equal-shot ties). These are bot comparisons,
not human win-rate estimates. Incoming shots never influence targeting, so paired
completion counts determine those races without needing to replay their turns.

### Final holdout results

On 160 previously unused random fleets (seed range starting at 400000), the final
implementation averaged **62.11 / 54.48 / 44.03 shots** for Easy / Medium / Hard.
Paired races gave Hard **117/160** wins over Medium and **134/160** over Easy;
Medium beat Easy **107/160** times. On eight rotations/reflections of a touching,
corner-packed fleet, averages were **43.50 / 44.12 / 40.38**. Individual fixtures
need not preserve difficulty ordering; the aggregate random-board separation is
regression-tested. Hard's slowest decision in this run was about 53 ms on this
machine, with other validation running concurrently (not a hardware guarantee).

Development comparisons rejected two tempting changes: applying sampled fleet
marginals to every early hunt, and a broad preference for cells near any sunk
ship. Both performed worse. The retained cluster preference requires multiple
sunk ships and dense geometric evidence. These results justify a stronger Hard
level, not a claim of mathematical perfection or guaranteed superiority to a
skilled human.
