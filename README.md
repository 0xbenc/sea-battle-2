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
Once both commanders are ready, aim and press **Enter** to fire. One shot per
turn, including hits; the first to destroy all seventeen enemy squares wins.
**Space/Enter** skips a result cinematic. **R** requests a rematch after the
battle; both commanders must agree. The first starter is random and alternates
on rematches. **F2** changes preferred HD/SD; **Q** leaves with confirmation.

Direct launch options:

```sh
./bin/sea-battle --host
./bin/sea-battle --join-file invitation.txt
./bin/sea-battle --host --port 4747 --advertise example.net:4747
./bin/sea-battle --host --no-map       # LAN / IPv6 / manual setup only
./bin/sea-battle --lookdev            # original 40-scene art showcase
```

The default host port is **TCP 4747**; `--port 0` chooses an available port.
`--advertise` adds a reachable endpoint to the invitation; it does not configure
a firewall or forward a port. With manual IPv4 setup, forward that external TCP
port to the host's listening port and permit it through the host firewall.
Global IPv6 still needs inbound firewall permission. A private router address or
CGNAT may require upstream configuration, switching hosts, reachable IPv6, or
playing on the same LAN. There is deliberately no relay fallback.

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

Real Internet/router interoperability and native Windows/macOS behavior still
need field testing; localhost sessions, simulated routers and cross-builds do
not prove those paths.

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
  No accounts, friends system, rendezvous service, relay, or required shared
  infrastructure. Invitations carry connection details rather than referencing
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
  paths automatically. Attempt temporary PCP, NAT-PMP, and UPnP router mappings,
  renew while hosting, and remove on exit (expiry handles unclean exits).
  Provide manual address/hostname and port overrides. A router mapping is not
  proof of internet reachability: report connection success only after the guest
  connects. Keep troubleshooting out of the normal Host/Join flow until needed.
- **Terminal sizing follows ASCII Bowl:** HD is 160 columns × 48 rows (primary);
  SD is 80 × 24. Fall back to SD when HD cannot fit, restore HD when space permits,
  use F2 to change the preferred size, and center the canvas in larger windows.
  Below SD, suspend local controls/animation and show resize guidance; the network
  session remains connected.
- **Frontload the graphics:** establish compelling art, readable action, and a
  presentation approaching a vertical slice before building the basic simulation.
  This follows the development approach of the sibling `../ascii-bowl` project.
- **Why this order:** Benjamin trusts the AI collaborator to handle familiar game
  simulation requirements; the early uncertainty is the visual direction, where
  his taste and art direction matter most. The presentation establishes the target
  for the underlying systems to grow into.
