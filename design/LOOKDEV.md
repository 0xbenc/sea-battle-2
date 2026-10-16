> The default executable now opens the live Host/Join game. Use `--lookdev`
> (or any scene/capture flag) for the scripted showcase documented here.
> Lookdev-specific simulation/network limitations below describe this review mode,
> not the live game; see [README](../README.md) for multiplayer controls.
> Live placement also adds twenty native horizontal tactical hull canvases.

# Operation Pest Tide — lookdev review

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
| F2 | Change preferred HD / SD size |
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
