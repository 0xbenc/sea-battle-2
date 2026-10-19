> The default executable now opens the live Host/Join game. Use `--lookdev`
> (or any scene/capture flag) for the scripted showcase documented here.
> Lookdev-specific simulation/network limitations below describe this review mode,
> not the live game; see [README](../README.md) for multiplayer controls.
> Live placement also adds twenty native horizontal tactical hull canvases.

# Operation: Unexpected Guests — lookdev review

This is a playable presentation laboratory, not the multiplayer game. It has 40
scripted scenes, both faction consoles, the ten agreed ships, native HD/SD art,
and complete shot vignettes. The sea is the world; the interfaces are equipment
composited over it. Human instruments use steel, brass, glass and needles. Alien
instruments use chitin, neural pulses and luminous grafts.

The accepted camera direction is restrained: top, side and front are deliberately
orthographic. No tiny oblique view is required to understand a ship. The aliens
are primarily organic, with limited technological implants. The cutscenes can be
absurd; their spectacle still represents a single ordinary Battleship shot.

## Run and review

From the repository root, with Go 1.26.3+ and the sibling `../dotmason` checkout:

```sh
make build
./bin/sea-battle --lookdev
```

The executable is `bin/sea-battle`. Assets are embedded; running the built binary
needs neither Dotmason nor Python, network access, or a server. Use a truecolor
terminal and a monospace font whose half blocks meet cleanly. HD is 160×48; SD is
80×24. Larger windows center the canvas, smaller windows fall back to SD, and
windows below SD suspend playback. Focus loss also suspends playback.

The default is a continuous tour. Tab opens the scene deck. Selecting a scene
stops the tour and holds its final frame until replayed. T restores the tour.

| Key | Action |
|---|---|
| Tab | Scene deck; arrows, Home/End and PgUp/PgDn select; Enter opens |
| Left / Right | Previous / next scene |
| Space | Pause / resume |
| R | Restart current scene |
| T | Toggle continuous tour |
| F | Toggle quarter-speed review |
| `.` / `,` | Pause and step forward / backward 0.1 seconds |
| V | Cycle top / side / front on single-view cinematic scenes |
| W | Cycle day / night / squall / spore tide outside tactical scenes |
| WASD | Move the tactical target or placement ghost |
| X | Rotate the destroyer placement ghost |
| Enter | Fire a scripted shot from either command station |
| B | Return to the corresponding command station |
| F6 | Save the current terminal cells as JSON in `captures/` |
| ? / Esc | Open / close help |
| Q / Ctrl+C | Exit and restore the terminal |

Start with `arrival`, then try `human-station` and `alien-station`. Fire a shot to
see launch → pursuit → impact/miss → confirmed plot. Targets deterministically
choose a demonstration outcome; they do not consult a hidden simulated fleet.
Use `human-fleet` and `alien-fleet` for recognition, `periscope` and `depth` for
alternate framing, and `overkill` / `digestive-event` for cinematic excess.
`bulletin` and `hivecast` establish the satirical voice.

```sh
./bin/sea-battle --list
./bin/sea-battle --scene exchange --size sd --tour=false
./bin/sea-battle --scene hive --camera front --weather 1 --paused
./bin/sea-battle --scene sinking --size hd --at 6.5 --dump ansi
./bin/sea-battle --scene hit --size sd --at 2.2 --dump json
```

## Artwork and composition

`internal/assets/fleet.mason` is the embedded Dotmason v1 asset. Its 31 designs
contain all ten ships in side/front/top views, native HD/SD inspection sizes,
additional tactical and formation sizes, and an eight-frame impact crown at
each resolution. There are 120 ship canvases plus two effect canvases (including the live game’s
horizontal tactical hulls). Runtime
lookup uses design and size names, not array order.

`tools/make_art.py` reproducibly authors the initial geometry at each native
size. It draws on each size's own grid and gates detail by resolution; it does
not shrink an HD bitmap to produce SD. `make art` intentionally regenerates the
file and will replace manual edits. Dotmason can inspect or edit the actual file:

```sh
../dotmason/bin/dotmason preview --view lod internal/assets/fleet.mason
```

Half pixels are composited before becoming terminal cells. Transparent samples
preserve the ocean underneath. The game owns clipping, reflections, weather,
lighting, particles, timing and camera cuts. The impact crown demonstrates actual
Dotmason animation playback. Its holds are interpreted locally as tenths of a
second. No `.mason` format or Dotmason source changes were needed.

Normal hostile grids contain confirmed shot marks and the current target only.
Inspection and cinematic scenes intentionally reveal vessels. Four fixed hulls
and a movable destroyer ghost exercise deployment appearance; the ghost checks
board bounds and occupied squares. This is not a full placement/game system.

## Self-review and corrections

The review used native terminal-cell images, temporal contact sheets, animated
SD captures, real PTY output, and deterministic renderer tests. Captures render
half blocks as exact upper/lower rectangles; they do not substitute higher
resolution artwork. The font renderer is an approximation of the user's terminal;
the PTY cell comparison verifies glyphs and RGB values independently of it.

| Finding | Revision |
|---|---|
| SD top-view hulls extended behind the console | Explicit camera safe areas keep complete inspection silhouettes visible |
| Too many clips shared one profile composition | Added formation views, bow-on views, periscope, underwater contact, flight operations and multi-cut shot sequences |
| Several aliens read as the same organism from above | Re-authored skiff, lurker and ravager silhouettes; differentiated front views and grafts |
| Large instrument rims collided with labels | Reduced dial footprint and gave HD stations a deeper instrument panel |
| Foreground splashes disappeared behind the SD console | Raised cinematic waterlines and placed misses in the visible foreground |
| A shot vignette ended without adding its result to the plot | Final sequence stage now shows the selected coordinate's hit/miss mark and status |
| Placement ghosts could appear valid on another ship | Bounds and occupied-square checks now drive the ghost color/status |
| Direct firing skipped the launch presentation | Enter now starts the complete faction-specific shot sequence |
| Shared RGB defaults could couple effect recoloring to ship paint | Gave foam its own palette identity and added a test preventing it from appearing in ship canvases |
| Sparse scene control feedback | Added exact capture feedback, quarter-speed review, stepping, and effective-camera/weather cycling |

The test matrix covers all 40 scenes at five times in both resolutions (400
frames), plus 96 camera/weather combinations. Additional tests cover transparent
compositing, determinism, hidden-fleet concealment, fleet coverage, placement,
sequence outcomes, aim bounds, clocks, tour transitions, focus and resize.

`tools/pty_review.py` drives the real executable through HD/SD switching,
centering, aiming, scene selection, help, undersize suspension, recovery,
animation, pause and quit. Its SD artwork region must match the deterministic
renderer cell for cell. It also verifies terminal attributes are restored.

```sh
make test
uv run --with pillow --with pyte tools/pty_review.py
make review
# Faster still-frame pass:
uv run --with pillow tools/review.py --quick
```

Generated captures and binaries are ignored. `captures/` holds native stills,
contact sheets, motion sheets, SD GIFs, and PTY evidence. The review script's
`--motion` accepts comma-separated scene IDs to regenerate a smaller movie set.

## Limits and next art decisions

The remaining approval is aesthetic: whether this level of chunky silhouette,
organic texture, color and absurdity is the right foundation. SD intentionally
simplifies detail; the tactical grid is for recognition and targeting, while the
inspection views show ship anatomy. The consoles contain animated illustrative
instruments, not real pressure or damage simulations.

There is no networking, complete match loop, saved game, audio, or real 3D camera.
Selecting scenes and firing examples do not accumulate a persistent match.
The final shot-sequence plot accepts another aim/fire command, so the visual
loop can be exercised repeatedly. The
Linux PTY is verified; native Windows/macOS terminal appearance still needs
hands-on review even when their builds succeed. The default tour is a catalogue,
not a claim that this is the final pacing or cutscene frequency of the game.


## Verification recorded on 2026-10-16

- `go test -race ./...` and `go vet ./...` passed.
- The Linux PTY review passed all 16 checks, including zero SD cell mismatches.
- Windows amd64 and macOS arm64 cross-builds succeeded; native terminal execution
  on those systems was not tested.
- An HD hit frame including ANSI encoding measured approximately 2.1 ms on this
  machine in the included benchmark. Actual terminal throughput can differ.
- Dotmason and ASCII Bowl source files were not modified. Their loader/runtime
  tests passed during the initial integration assessment.

## Title-screen idle direction

A session-owned director now replaces the fixed-period loops. Play supplies a
fresh random seed at startup; captures use explicit seeds. Story selection,
variants, and wait times are drawn once by the director, never by rendering.
Resizing, extra View calls, and frame drops cannot reroll a story. Scheduling uses
bounded state and handles elapsed-time jumps in chronological order.

The opening waits 5–10 seconds. Major stories run to completion, followed by
10–20 seconds of quiet before the next story. Each shuffled round plays all nine major events once before any repeats.
Round boundaries also prevent immediate repeats, and dogfight/bombing sorties
never run consecutively. Ambient
shoreline bursts, radar and defensive fire have separate 35–75-second waits and
are moved into quiet gaps so they cannot interrupt or obscure a major story.
Title glints also wait 35–75 seconds between sweeps.

The story pool:
- Jet sortie: deck doors open, an elevator raises the jet, then launch, distant
  pass, pursuit, and destruction or a landing. A surviving aircraft arrests on
  deck, lowers into the hangar, and the doors close; no jet idles on the deck.
  Takeoff has a brief tension beat, a fast accelerating catapult stroke, and
  lingering deck steam; climb begins as the jet clears the bow.
  Curved pursuit paths preserve velocity through the banked reversal. Landing
  uses a steady approach, flare, visible gear, and a decelerating arrested rollout.
  Some pursuits climb behind the title's actual letter mask.
- Distant bombing: the sortie returns as a tiny aircraft, follows one broad curved
  swoop beneath the subtitle, drops five bombs on
  shoreline cells, and comes home. This replaces the pursuit entirely.
- Brood missile sub: purple chitin and mint biotech surface, launch three arcing
  organic missiles into shoreline targets, then submerge.
- Hostile resupply: helicopter winches down a crate and leaves. After a beat,
  the crate waits an extra three seconds, opens its eyes, sprouts legs, scuttles across deck and leaps overboard.
- Flare escape: a damaged jet drops a fan of flares. The pursuer abandons the
  aircraft to bite one, with predictable consequences.
- Bad landing: a violet troop sac skips three times across the sea, then crashes
  into the beach.
- Shore awakening: a chitin cannon unfolds from the land, charges, and lobs a
  glowing projectile over the carrier into an American escort entering from the
  right in the midground. Impact hits its superstructure; secondary explosions,
  fire, smoke, and a bow-down list follow.
- Leviathan: a luminous spined back briefly rises behind the carrier.
- Popup CIWS: a hooked chitin drone dives into a continuous sweeping turn, its body
  following the flight direction without hovering at the turnarounds.
  A split hatch opens and the radar-topped rotary gun snaps up. Three tightening
  tracer bursts, recoil, muzzle flash, and ejected brass lead into armor hits,
  dismemberment, tumbling wreckage, and a splash. The gun cools and retracts.

Live deployment uses a separate own-waters composition: a 40×30 half-pixel board
at SD and 80×60 at HD, with a compact faction-tinted fleet rack. Ship-shaped
previews show valid/blocked positions, then disappear for the complete-fleet
review. Only one contextual coaching line remains; extended controls live in
the `?` panel. Mouse and keyboard share placement, edit, undo, shuffle, and launch
actions. Pending/locked fleets cannot be edited. `tools/placement_pty_review.py`
checks manual mouse placement through launch for both factions and sizes.

Live combat gives hostile waters the large deployment-sized chart and keeps your
fleet on a smaller chart beside a hull-cell damage rack. Tab or a small-chart
click swaps focus; fleet-name clicks inspect the corresponding friendly ship.
Inspection never fires or silently moves the saved target. Row/column highlights,
cell-contained foam ripples, flame sprites, and submerged wreck markers, distinct marked-cell fills, and the latest-shot border make
coordinates and outcomes readable at both sizes. A single coaching line handles
aiming, resolved cells, waiting, connection loss, and final accuracy; detailed
controls live behind ?. SD's small friendly chart uses bright continuous tactical
hulls with dark outlines instead of reducing detailed art until it disappears.
Miss foam and hit flames have a slow, low-amplitude traveling highlight; marker
footprints stay fixed. Solo shot handoff fades the outgoing cinematic to black
and the incoming cinematic back in over two seconds, with no aiming modal.
The fade cannot be skipped into an accidental shot; timed result reveal remains.
Live shot cinematics are full-screen effects with one large pixel-art result:
YOU MISSED! / THEY MISSED!, YOU HIT! / YOU WERE HIT!, or SUNK!.
Only a sinking adds the ship name. Tactical headers, coordinates,
instrument panels, shooter banners, and the control footer stay off the cutscene;
Space/Enter still skips it. The lookdev showcase retains its instrumentation.

Match endings have eight compositions across faction, victory/defeat, and margin.
Close victories show a scarred surviving hull, smoke, fire, and floating debris.
Overwhelming human victories have escorts and a formation flyover; overwhelming
brood victories add spores and bioluminescent tendrils taking over the water.
Defeats shift the camera to a sinking friendly wreck with the victor behind it;
overwhelming defeats add multiple wrecks and the occupying fleet. Each has one
short faction-specific joke, remaining winner HP, and mouse/keyboard rematch,
chart review, menu, and quit controls. Endings start after the final shot resolves,
and rematches reset their presentation state. No enemy placement is revealed.

The overwhelming threshold is 9/17 unhit hull cells. A fixed-seed sample of 144
bot matches (48 each: medium/medium, hard/medium, medium/easy) produced 7 such
winners; this is calibration evidence, not a prediction of human win rates.
`go run ./tools/result-eval` reproduces the comparison. `TestEightResultScenes`
captures all eight endings at four animation times in both sizes when
`SB2_CAPTURE_DIR` is set. Controller tests play actual winning shots from both
perspectives, then verify final-cinematic gating, chart review, and rematch reset.
Mouse press fires through the same authoritative gates as Enter; release,
inspection, resolved cells, pending shots, and enemy turns cannot fire.
`tools/combat_pty_review.py` exercises these interactions with real terminal mouse
events for both factions at SD and HD. Renderer tests cover results, disconnection,
help, chart focus, and concealment of enemy hulls.

The ocean remains in motion during quiet intervals. Far events render behind the
carrier; deck action renders in front; titles and controls render last. The black
translucent control panel keeps plain white shortcut letters and centered quit.
SD/HD follow terminal size automatically; `--size sd` supports deliberate review.

For exact SD/HD story captures and optional SD motion frames:

```sh
SB2_CAPTURE_DIR=captures/random-stories SB2_MENU_MOTION=1 go test ./internal/visual -run 'TestIdleStoriesInContext|TestCarrierFlightReview|TestCaptureMenuMotion' -count=1
```
