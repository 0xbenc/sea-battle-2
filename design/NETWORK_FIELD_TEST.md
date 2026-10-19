# Cross-network connection field test — 2026-10-19

Two real machines, in different US states: Linux/WSL2 and an Apple Silicon Mac
running macOS. Starting source commit on both: `2b81d15c40d405433267d099e0a34b70cab74683`.
Cloudflare connector version: `2026.10.1`. Tests used the actual `session.Controller`, pinned TLS handshake, invitation
credential, fleet deployment, turn validation, shots, and final authoritative
state. The automated results below are headless matches. A subsequent human playtest
is recorded separately below.

## Results

| Route | Host → guest | Result | Guest authentication / complete match |
|---|---|---|---|
| Automatically discovered Tailscale routes | WSL2 → Mac | PASS, 182 shots | 263 ms / 6.700 s |
| Automatically discovered Tailscale routes | Mac → WSL2 | PASS, 182 shots | 307 ms / 7.945 s |
| Public WSS / Cloudflare Quick Tunnel | WSL2 → Mac | PASS, 182 shots | 880 ms / 9.228 s |
| Public WSS / Cloudflare Quick Tunnel | Mac → WSL2 | PASS, 181 shots | 1.015 s / 10.329 s |
| PCP / NAT-PMP | WSL2 gateway | Unavailable | UDP replies timed out |
| PCP / NAT-PMP | Mac gateway | Unavailable | UDP connection refused |
| UPnP | Both gateways | Unavailable | No usable short-lived mapping |
| Public native IPv6 | WSL2 as client | Unavailable | WSL2 has no public IPv6 route |

The random starting player accounts for 181 versus 182 shots. Both endpoints
reported PASS for each completed match. Public tests replaced the guest's
initial invitation routes with **only** the public WSS URL; the guest reported
that WSS URL as its authenticated route. Tailscale was used to administer the
machines and transfer private invitation files, but was not the public test's
game transport. Tailscale itself can use direct or relayed underlay paths; these
tests do not establish which underlay it used.

The original interactive failure was not captured, so it cannot be assigned a
single proven cause. What is established: TCP/TLS gameplay works across these
machines; neither tested gateway supplies an automatic public mapping; WSL2
cannot dial the Mac's public IPv6 addresses. An invitation containing only
unreachable LAN/router routes cannot fix those network constraints.

## Playing now: public route, no shared VPN or account

The host needs Python 3, the updated game, and the `cloudflared` binary. Both
players need the updated game to understand WSS invitation endpoints. These
binaries have been placed in each checkout's `bin/` for this field test.

Host, from the game checkout:

```sh
python3 tools/host-public.py
```

The launcher opens a temporary public tunnel, waits for it to register, and
starts the game as host. Use **C** to copy the full invitation (or **S** to save
it privately). Send the invitation privately to the guest. The guest runs:

```sh
./bin/sea-battle
```

Then **Join**, paste the full invitation, and Enter. The guest needs neither
Cloudflare software nor a Cloudflare account nor Tailscale. Keep the host game
running. Exiting the game also stops the launcher's tunnel and deletes its
temporary logs. Restarting the launcher creates a new URL and game session;
share its new invitation. `CLOUDFLARED=/path/to/cloudflared` selects an existing
installation. Extra launcher arguments go to the game, e.g. `--size sd`.

For a fresh checkout, `make build` builds the game; obtain `cloudflared` from the
[official releases](https://github.com/cloudflare/cloudflared/releases) for the
host OS and put it at `bin/cloudflared` (executable), or set `CLOUDFLARED`.
Nothing is installed as a system service and no Cloudflare login is performed.

### What Cloudflare does

```text
guest game -- HTTPS/WebSocket --> Cloudflare public URL
                                      |
                          host's outbound cloudflared tunnel
                                      |
                         127.0.0.1:<HTTP port>/sb2
                                      |
                             host game TLS handler
```

The public HTTPS layer terminates at Cloudflare. Inside the WebSocket stream,
the game performs a separate TLS 1.3 handshake pinned to the exact certificate
fingerprint in the private invitation. Its bearer key, fleet state, and commands
remain inside that end-to-end encrypted stream. Cloudflare can observe traffic
metadata and availability depends on its service. Knowing the public URL alone
does not grant a game seat. The local HTTP listener binds only to loopback and
serves only `/sb2`; it is not a general TCP proxy or filesystem server.

[Quick Tunnels](https://developers.cloudflare.com/tunnel/get-started/quick-tunnels/)
provide temporary addresses without an account. They are intended for testing,
not a production availability guarantee. Cloudflare documents
[WebSocket support](https://developers.cloudflare.com/cloudflare-one/faq/cloudflare-tunnels-faq/).
There is no automatic third-party tunnel in the normal Host flow. The public
launcher is explicit and optional. A long-lived production relay would need a
separate hosting/reliability decision.

Manual equivalent (two host terminals):

```sh
./bin/cloudflared tunnel --url http://127.0.0.1:8787 --no-autoupdate
# Copy the generated hostname into this command:
./bin/sea-battle --host --no-map --websocket-port 8787 \
  --advertise wss://GENERATED-NAME.trycloudflare.com/sb2
```

## Playing over Tailscale

Both players must be able to reach each other through Tailscale. In this setup
it runs inside WSL2, so the game can bind its interface directly. A Tailscale
installation on Windows alone does not automatically expose a WSL2 listener.

Host with `./bin/sea-battle --host --no-map`, then share the full invitation.
Interface discovery includes the VPN addresses. If needed, explicitly advertise
`--advertise HOST-TAILSCALE-IP:4747`. The guest follows the same Join/paste flow.
No router changes are needed for the tested Tailscale route.

## Changes and guarantees

- Invitation routes can now include `wss://host[:port]/sb2`. TCP and WSS routes
  race concurrently; every successful candidate must satisfy the original host
  certificate pin. Failed candidates retain endpoint-specific error details.
- Optional `--websocket-port` exposes the game's TLS handler through a loopback
  HTTP listener. Connections share the normal admission limit, authentication,
  seat ownership, reconnect semantics, and server cleanup.
- `--connect-address HOST:PORT` (or a WSS URL) adds a guest-side route without
  replacing the invitation's identity. It survives host route refreshes and
  reconnects; useful for an independently configured tunnel/forward. Other
  invitation routes remain candidates. It requires `--join` or `--join-file`.
- Forced teardown sets a deadline before closing, including nested WebSocket
  streams, so TLS/WebSocket close frames cannot indefinitely block cleanup.
- Router discovery now reports which mapping methods failed.
- No game rule, fleet privacy, or certificate verification was weakened. No
  operating-system firewall, router admin setting, SSH configuration, Tailscale
  configuration, or unrelated Mac checkout was changed.

## Reproducing protocol tests

```sh
go build -o bin/netcheck ./cmd/netcheck
# Host; writes a mode-0600 invitation, never prints the credential:
./bin/netcheck --host --invitation bin/test.invite --timeout 90s
# Transfer test.invite privately to the guest, then there:
./bin/netcheck --invitation bin/test.invite --timeout 60s
```

For public-route tests add `--websocket-port 8787 --advertise wss://HOST/sb2`
to the host and `--only-route wss://HOST/sb2` to the guest. Start the host's
Cloudflare tunnel as above. The authenticated route is printed. `--only-route`
constrains the initial attempt; subsequent authenticated host route updates may
add routes for reconnect. No disconnection was induced during the public field
matches. Reconnect and route preservation are additionally covered by automated
tests; that is distinct from a field-tested tunnel outage recovery.

`--map` enables the same short-lived router mappings as the game. Its invitation
file follows host route updates atomically. Remove the test invitation after
both processes finish; existing files are deliberately not overwritten at startup.
A timed-out mapping-only probe exits nonzero because no guest played a match.

Validation includes `make test` (all Go tests and vet), race tests for peer,
session and router code, and native macOS peer/session/router test binaries.
New regressions exercise WSS inner certificate pinning, WebSocket shutdown,
listener cleanup, mixed failing/successful endpoints, safe endpoint diagnostics,
and an explicit route surviving a host update followed by reconnect.

## Human playtest

After the automated checks on 2026-10-19, Benjamin hosted from WSL2 with
`python3 tools/host-public.py` and Vik joined from macOS through the normal game
flow. Benjamin reported that they played successfully and that it worked great.
He also noted that the screens need cleanup. This validates the interactive
cross-network play flow in addition to the automated protocol matches.

The separate launcher remains a temporary workflow. Integrating the optional
public tunnel into the normal Host / Join flow and cleaning up the screens are
follow-up work; this change does not claim to complete either.
