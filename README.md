Greetings Program!

Sea Battle 2 is a multiplayer recreation of Hasbro’s Battleship board game, played
in the terminal.


Run `make run` for the live **Host / Join** menu (Go 1.26.3+ and the sibling
`../dotmason` checkout are needed to build; `bin/sea-battle` is self-contained).
The host commands the Americans; the guest commands the brood.

**Host → C copy invitation → send it privately → Join/paste.** If the terminal
doesn't support clipboard copying, **S** saves the complete invitation to a
private `invitation.txt` file. The on-screen preview is deliberately abbreviated.
When automatic router mapping finishes, copy the updated invitation. The host
automatically enters deployment when the guest connects. **Enter** lets the host
deploy early; **I** reopens the invitation console while hosting.

Place ships with **arrows/WASD**, **1–5** selects a hull, **X** rotates, and
**Enter** places/repositions it. **Backspace** lifts the selected hull.
**R** arranges a valid random fleet; **Space** locks it in. Ships may touch.
Once both commanders are ready, hover or use **arrows/WASD** to aim on the large
hostile chart; **click/Enter** fires. **Tab** or clicking the small chart switches
between targeting and inspecting your fleet without firing. Click a fleet name
to inspect that ship. Highlighted axes track your aim; foam ripples mark misses,
bright flames mark hits, and submerged metal marks sunk hull cells. The fleet rack shows surviving hull cells,
and each chart labels its latest shot. **?** opens the compact combat help panel.
One shot per
turn, including hits; the first to destroy all seventeen enemy squares wins.
**Space/Enter** skips a result cinematic. **R** requests a rematch after the
battle; both commanders must agree. The first starter is random and alternates
on rematches. **Q** leaves with confirmation.

Finished battles have eight animated endings: close or overwhelming victory and
matching defeat scenes for each faction. **Overwhelming means the winner retains
at least 9 of 17 unhit hull cells**; 1–8 gives the close result. Both commanders
see the same classification, calculated from confirmed damage. **Tab** switches
between the ending and final charts; **R** requests a rematch.

Direct launch options:

```sh
./bin/sea-battle --host
./bin/sea-battle --join-file invitation.txt
./bin/sea-battle --host --port 4747 --advertise example.net:4747
./bin/sea-battle --host --no-map       # LAN / IPv6 / manual setup only
./bin/sea-battle --lookdev            # original 40-scene art showcase
```


For players on different networks without a shared VPN, the host can run
`python3 tools/host-public.py` for an optional public tunnel, then share the
normal full invitation. Only the host needs `cloudflared`; no Cloudflare account
is used. The guest uses the usual Join menu. See [setup, privacy, limitations,
and verified results](design/NETWORK_FIELD_TEST.md).

The default host port is **TCP 4747**; `--port 0` chooses an available port.
`--advertise` adds a reachable endpoint to the invitation; it does not configure
a firewall or forward a port. With manual IPv4 setup, forward that external TCP
port to the host's listening port and permit it through the host firewall.
Global IPv6 still needs inbound firewall permission. A private router address or
CGNAT may require upstream configuration, switching hosts, reachable IPv6, or
playing on the same LAN. An optional public WebSocket tunnel is available when direct routes fail; see
[the cross-network setup and field tests](design/NETWORK_FIELD_TEST.md).

Invitations expire after 24 hours and contain a private bearer key plus the
host certificate fingerprint. Keep them private. Clients try the supplied
addresses directly using TLS 1.3 and exact certificate pinning. The host owns the
rules and sends each player only their own fleet and confirmed shot information;
an unsunk enemy hit reveals no ship identity. This is a trusted-friends game:
the hosting process necessarily holds both fleets, so this is not an anti-cheat
boundary against the host.

Detected connection loss pauses firing and triggers automatic reconnect. The host retains
the match in memory; reconnecting resynchronizes it without replaying commands or
old cutscenes. Queued commands are bound to their original connection and cannot
fire after reconnecting. A per-client resumption key lets the same client replace
a half-open socket; an independent client cannot displace an occupied seat merely
by pasting the same invitation. Closing/restarting the host ends the session. There is no save/load
or host migration yet. Router mappings request 120-second leases, renew while
hosting, and are removed on shutdown where possible; expiration covers crashes.
A failed renewal is retried while the last confirmed lease remains valid.
Expired mappings trigger rediscovery; router and local-interface address changes
refresh the invitation. Local address refresh also runs with `--no-map`.

Connected guests receive updated routes through their existing pinned TLS
connection. If every route breaks before that update arrives, the host shares
the current invitation again; the guest presses **J** while disconnected and
pastes it to update routes for the same session. This preserves the match and
rejects invitations for a different host/seat. No direct-only system can deliver
a new address over a connection that no longer exists.

The main menu offers **? Help**, **T Settings**, and **Q Quit**. All six main-menu
controls support left-click and a subtle mouse-hover highlight. Clicking clear
water makes a quick ripple (two-second cooldown); carrier clicks spark and title
letters glint. Rocks, moving objects, and controls are excluded from water clicks.
Settings include
Auto/SD display, animated/still menu, and default solo difficulty. Preferences
are saved in the OS user config directory under `sea-battle-2/settings.json`;
explicit `--size` and `--difficulty` options override saved preferences.
Single-player selection uses separate translucent faction panels, native fleet
portraits in HD, and clickable difficulty choices. Team cards briefly flash
before deploying; Back cancels a pending selection.

The **40-scene lookdev showcase** remains available with `--lookdev`.
**Tab** opens its scene deck; **Space** pauses, **R** replays, **F** slows motion.
Its tour and shot exchanges are scripted presentation studies, separate from
live play. See [the lookdev guide](design/LOOKDEV.md) for art workflow and controls.

Verification: `make test` runs rules, TLS matches/reconnect/privacy, simulated
router leases, UI gates, renderer checks, and vet. `go test -race ./...` checks
concurrency. `tools/play_pty_review.py` exercises two real terminals; the original
`tools/pty_review.py` checks lookdev. Run either with
`uv run --with pillow --with pyte` after `make build`.
The hardening gate is `make hardening`: race checks, vet, repeated failure/rematch
tests, and three count-bounded fuzz targets. `make test-terminal` runs both PTY
reviews. See [the hardening record](design/HARDENING.md) for ownership, guarantees,
failure scenarios, and validation evidence.

Real WSL2 ↔ macOS matches have passed over Tailscale and an optional public
WebSocket tunnel in both hosting directions. Router mapping was unavailable on
these networks; native Windows and other router interoperability remain unproven.
See [the field test record and launch instructions](design/NETWORK_FIELD_TEST.md).

The two teams use the same fleet lengths, with distinct ship identities. Team A
is the original American fleet from `0xbenc/SeaBattle`; Team B is an alien naval
fleet with a tongue-in-cheek, Klendathu-inspired insect-war theme.

| Spaces | Team A — Americans | Team B — Aliens |
|---|---|---|
| 2 | Cyclone Class Patrol Boat | Drone Skiff |
| 3 | Virginia Class Attack Submarine | Deep Lurker |
| 3 | Freedom Class Littoral Combat Ship | Littoral Ravager |
| 4 | Zumwalt Class Destroyer | Siege Carapace |
| 5 | Gerald R. Ford Class Aircraft Carrier | Hive Carrier |

Art presentation follows these additive principles:

- **The ocean is the playspace:** depict vessels operating at sea, not a physical
  plastic Battleship board or its upright divider. This sets the visual framing;
  it does not replace the Battleship rules.
- **Two faction-specific control overlays:** composite human naval controls for
  Team A and alien controls for Team B over the ocean. Each faction gets its own
  visual language for the player's controls.
- **Diegetic where practical:** favor instruments that belong to the faction's
  world (a pressure gauge rather than a generic progress bar) when the function,
  design, and available terminal cells/half pixels support it. Preserve clear
  meaning and usability in both HD and SD; do not force an instrument metaphor
  where it does not fit.
- **Drama and effects:** make room for dramatic moments and cutscenes with FX.
  Specific sequences and their pacing are explored in the lookdev showcase.
- **Tone:** Starship Troopers-style humor and tongue-in-cheek human/bug hostility
  belong alongside the naval spectacle. A more tangible ocean setting does not
  imply a humorless simulation.
- **Camera discipline:** use top, side and front views; any angled view must earn
  its readability at the actual terminal resolution, especially SD.
- **Alien materials:** mostly organic, Zerg-like nastiness with a little Borg-like
  grafted technology. Cutscenes may go completely over the top while the gameplay
  stays classic Battleship.

- **Multiplayer is directly player-hosted:** Host → copy invitation → Join/paste.
  No accounts, friends system, rendezvous service, or required shared
  infrastructure for direct play. An explicit public-tunnel launcher provides an
  optional fallback through Cloudflare Quick Tunnels. Invitations carry connection details rather than referencing
  a centrally stored game code.
- **Minimize network setup, without promising universal connectivity:** automate
  local address discovery and router port mapping where supported; support LAN,
  IPv6, and manual address/port configuration. Explain connection failures and
  offer actionable setup guidance. Some networks will require manual setup or
  swapping which player hosts. Do not depend on public STUN or IP-lookup services.
- **Implemented networking design:** Go TCP + TLS on one
  listening port. Self-contained invitations carry candidate addresses, port,
  protocol version, host certificate fingerprint, and a random session secret;
  no certificate authority is required. Try LAN, global IPv6, and mapped IPv4
  paths automatically, including VPN interfaces. Optional WSS invitation routes
  carry the same pinned TLS session through a public HTTP tunnel. Attempt temporary PCP, NAT-PMP, and UPnP router mappings,
  renew while hosting, and remove on exit (expiry handles unclean exits).
  Provide manual address/hostname and port overrides. A router mapping is not
  proof of internet reachability: report connection success only after the guest
  connects. Keep troubleshooting out of the normal Host/Join flow until needed.
- **Terminal sizing follows ASCII Bowl:** HD is 160 columns × 48 rows (primary);
  SD is 80 × 24. Fall back to SD when HD cannot fit, restore HD when space permits,
  and center the canvas in larger windows. The in-game size toggle is removed;
  `--size sd` remains available for deliberate SD review.
  Below SD, suspend local controls/animation and show resize guidance; the network
  session remains connected.
- **Frontload the graphics:** establish compelling art, readable action, and a
  presentation approaching a vertical slice before building the basic simulation.
  This follows the development approach of the sibling `../ascii-bowl` project.
- **Why this order:** Benjamin trusts the AI collaborator to handle familiar game
  simulation requirements; the early uncertainty is the visual direction, where
  his taste and art direction matter most. The presentation establishes the target
  for the underlying systems to grow into.

## Single player

Press **S**, choose **1 Easy / 2 Medium / 3 Hard**, then **A** for the American
navy or **B** for the brood. Medium is the default. The bot
commands the other fleet, using only revealed shot results. No networking setup
or connection is needed. Deployment shows only your own waters, with a larger
board and a compact fleet list. Move the mouse or use arrows, then click/Enter
to place; the next unplaced ship is selected automatically. Right-click/**X**
rotates, **1–5/Tab** selects, **Delete** removes, and **U** undoes. Click a placed
ship to move it; **Esc** cancels an unfinished move. **?** opens the control guide.
Once all five ships are placed, review and **Space** to launch. **R** shuffles the
fleet for a quick start. Locked fleets wait on this same own-waters screen.
Both factions use the existing rules, consoles and skippable shot cinematics.

`./bin/sea-battle --solo american --difficulty hard` starts deployment directly;
use `--solo alien` for the brood. Easy follows hits without sophisticated search;
Medium eliminates impossible targets and follows aligned hits; Hard infers complete
fleets, searches systematically and solves small endgames.
All three play fairly, and shot playback reveals board changes in sequence.
**R** after the result starts a rematch with the opening turn alternated.
**Esc** returns to the menu; unfinished matches ask before discarding progress.
See [single-player design and validation](design/SINGLE_PLAYER.md).
