# Automatic multiplayer connections

Normal **Host / Join** now owns the whole connection lifecycle. There is no
special launch command, Python dependency, or player approval during negotiation.
Run `make run` on each machine, choose Host on one and Join on the other, and
share the complete private invitation as usual.

## What happens

1. Choosing Host starts the game listener, local router discovery, and an
   anonymous Cloudflare Quick Tunnel. The menu and solo play start no connector.
2. The game waits for the connector to register and makes a bounded check of the
   public WebSocket endpoint before offering its invitation. A failed host-side
   check does not discard the registered tunnel: the guest may have working DNS
   and egress even when the host cannot reach its own public URL. The host can
   arrange a fleet while setup runs. Empty invitations cannot be copied or saved.
3. The guest contacts the public route first. If that route is unavailable,
   reachable direct endpoints remain a fallback.
4. After public authentication, the guest immediately probes the direct routes
   advertised by the host in parallel. These include LAN, IPv6, VPN interfaces,
   manual addresses, and any router mapping already discovered. The probe window
   is bounded; there is no player confirmation or settings dialog.
5. A successful candidate must pass the exact same certificate pin and a
   credential/resumption-key check. Probing never claims or displaces a seat.
6. The chosen connection activates the existing seat and receives its current
   authoritative state. The guest acknowledges it; the host confirms the selected
   transport. The guest exposes gameplay controls only after this confirmation.
7. On a confirmed direct connection, the host removes the public URL from the
   invitation and shuts down its connector process. If direct probes fail, the
   public stream remains the final game connection. It closes with the session.

"Direct" here means the game's TCP route instead of its Cloudflare WebSocket
route. A Tailscale route may itself use a relayed underlay; the game does not
claim to distinguish that. This does not implement UDP hole punching, WebRTC,
or a new publicly hosted rendezvous service.

Only the host needs a connector. The guest needs only the game. Cloudflare sees
connection metadata; game credentials, fleets, and commands remain inside the
separate end-to-end TLS 1.3 connection pinned by the invitation. No Cloudflare
account or login is used. Quick Tunnels remain a third-party testing service,
not an availability guarantee.

## Helper ownership and first use

The host looks for `CLOUDFLARED`, a helper beside the executable, `bin/cloudflared`,
or an existing PATH installation. If absent on a supported platform, it downloads
Cloudflare's pinned `2026.10.1` release, verifies its published SHA-256 digest,
and installs it atomically into the game's OS-specific user cache. Supported
automatic downloads cover Linux amd64/arm64, macOS amd64/arm64, and Windows amd64;
only Linux/WSL2 and macOS were exercised in this field test.

The game does not install a service or change firewall, SSH, Tailscale, or system
DNS settings. It uses a temporary empty connector configuration and strips
inherited `TUNNEL_*` variables so unrelated Cloudflare accounts/configuration do
not become part of this anonymous session. Temporary configuration is removed
on shutdown. Normal cancellation joins setup workers and connector processes,
with a bounded forced termination if graceful shutdown stalls.

If helper setup fails, a direct invitation is still offered and public setup is
retried. Advanced `--no-public` disables this optional service entirely. Existing
manual `--websocket-port`/WSS `--advertise` configurations remain externally owned;
the game cannot stop a connector it did not launch. `tools/host-public.py` is now
only a compatibility wrapper around the ordinary game host command.

## DNS readiness

Field tests exposed stale negative DNS answers for newly allocated Quick Tunnel
hostnames. Registration alone was not proof that a guest could resolve the name.
The host now attempts an endpoint readiness check before publishing the
invitation, while retaining the guest as the authority on actual reachability.

For a single-label `*.trycloudflare.com` hostname only, a failed ordinary DNS
lookup can race Cloudflare and Google HTTPS DNS endpoints. The original hostname
is preserved for HTTPS SNI, certificate validation, and the WebSocket Host header.
The inner game certificate pin is unchanged. The DNS providers see the requested public hostname; no invitation token is
included in the DNS query. Arbitrary invitation hosts do not use this fallback; response size,
address count, and request durations are bounded.

References: [Quick Tunnels](https://developers.cloudflare.com/tunnel/get-started/quick-tunnels/),
[WebSocket support](https://developers.cloudflare.com/cloudflare-one/faq/cloudflare-tunnels-faq/),
and [Cloudflare HTTPS DNS](https://developers.cloudflare.com/1.1.1.1/encryption/dns-over-https/make-api-requests/),
and [Google HTTPS DNS](https://developers.google.com/speed/public-dns/docs/doh).

## Recovery and compatibility

The host still owns the match, and queued commands remain bound to their original
socket. A normal direct disconnection reconnects using the same resumption key
and current invitation. Neither shots nor fleet commands are replayed by handoff.
Failed direct probes leave the public connection intact. If activation fails,
the client retains the public stream when possible or reconnects through the
still-running public connector. An unconfirmed direct socket never authorizes
public shutdown.

After a direct connection has replaced the public route, a prolonged loss starts
a new public connector on the host. The host keeps the same match, fingerprint,
and seat credential. **If every previous route is gone, the host must share the
updated invitation and the guest uses J to paste it.** A restarted Quick Tunnel
has a new URL; without a surviving connection or a separate stable rendezvous
service, the new URL cannot be delivered automatically. The same limitation
applies when a public connector crashes and gets a new URL. Brief direct
interruptions normally reconnect before the delayed public restart is needed.

Negotiation is advertised as an optional wire capability. New clients send
control messages only when the server advertises support; old peers can retain
their original behavior. Both players should update for the complete automatic
handoff. Protocol version and invitation identity are unchanged.

## Validation

Automated regression coverage includes public-first bootstrap, direct promotion,
public retention when probes fail, public-service failure with direct fallback,
failed activation without losing the working public socket, cancellation during
probes/setup, occupied-seat protection, route removal and connector retirement,
listener cleanup, verified downloads, and DNS fallback boundaries. The original
complete-game, reconnect, privacy, rule, renderer, and terminal tests also run.

The first real automatic handoff tests between WSL2 and Vik's Mac completed full
matches in both directions. Public-only WSL2 hosting also completed a match. A
subsequent Mac-hosted public-only test initially failed on negative DNS results;
that failure prompted the readiness and HTTPS DNS changes above rather than
being counted as a successful test.

The final results below use the completed negotiation and DNS fallback code.

### Final cross-machine results — 2026-10-19

Both machines ran native builds of the new code. Every row below completed an
entire match and produced PASS at both endpoints. Guest times include connection
establishment; host startup time is excluded from those guest measurements.

| Host → guest | Verified initial / final route | Shots | Guest connected / match complete |
|---|---|---:|---|
| WSL2 → macOS | Public WSS → direct Tailscale IPv6; connector stopped | 182 | 1.515 s / 12.614 s |
| macOS → WSL2 | Public WSS → direct Tailscale IPv6; connector stopped | 182 | 1.648 s / 14.542 s |
| WSL2 → macOS | Public WSS retained after failed direct probe | 181 | 1.002 s / 11.090 s |
| macOS → WSL2 | Public WSS retained after failed direct probe | 181 | 0.840 s / 11.692 s |

Public-only cases advertised an intentionally closed direct endpoint through the
netcheck diagnostic override. No firewall or routing settings were changed to
force these cases. Both handoff cases explicitly logged the initial public
bootstrap, the selected direct route, and connector shutdown after confirmation.

A separate live Mac-hosted match deliberately broke the WSL2 guest's ordinary
DNS resolution in the test dialer. The HTTPS DNS fallback preserved verified WSS
and pinned inner TLS, completing **182 shots in 12.02 seconds**. It recorded four
forced ordinary DNS failures. This test is opt-in:

```sh
SB2_FIELD_INVITATION=private-invitation-file \
  go test ./internal/peer -run '^TestFieldPublicDNSFallback$' -v -count=1
```

A real-terminal cross-machine check launched both native games with **no game
arguments**. It selected Host and Join from the default menus, copied/pasted the
invitation, reached deployment and battle on both machines, confirmed the same
shot on both charts, and exited both terminals cleanly. The ordinary local PTY
review also passed reconnect, shot history, resizing, and terminal restoration.
This is automated terminal coverage, not a claim of another human playtest.

A fresh-cache test hid all existing connector installations. The game downloaded
and verified its pinned helper automatically, opened a public route, and cleaned
up when the test deadline ended. Local full-suite race tests and vet passed;
native macOS peer, session, and connector test binaries also passed. Security
regressions verified rejection of both an incorrect inner pin and an untrusted
outer HTTPS certificate, including the DNS fallback path.

Useful diagnostics (not required to play):

```sh
go build -o bin/netcheck ./cmd/netcheck
./bin/netcheck --host --invitation bin/test.invite
# Guest, after privately transferring the file:
./bin/netcheck --invitation bin/test.invite
# Host variant to exercise public retention with a failed direct candidate:
./bin/netcheck --host --direct-routes 127.0.0.1:1 --invitation bin/test.invite
```

Do not reuse a pre-existing test invitation filename; remove test credentials
after the processes exit. The normal game uses its existing private copy/save
controls. No separate launcher is needed on either machine.
