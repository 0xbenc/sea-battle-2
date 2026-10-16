# Direct-session hardening

This pass covers connection/router recovery, session ownership, and adversarial
timing tests. It preserves the classic game rules, faction artwork, SD/HD sizing,
and direct player-hosted architecture. There are no shared networking services.

## Ownership and lifecycle

`internal/session.Controller` owns each session operation: host listener, client,
router discovery/renewal, local-address refresh, and shutdown. Bubble Tea owns
presentation and sends intents through this controller.

Operations have monotonically increasing generations. Starting a replacement
cancels the old operation and joins its cleanup before binding the new listener.
Events and asynchronous command results carry their generation; stale results
cannot modify the new presentation. Commands also capture the actual transport
connection before the UI schedules their write. A connection change invalidates
that intent, even if the session and board have not changed.

Lifecycle states are idle, starting, connecting, active, reconnecting, failed,
stopping, stopped, and closed. Active means the local connection has received an
authoritative state; opponent presence and match phase remain separate facts.
`Stop` is asynchronous. `Close` is idempotent, cancels work, and joins all owned
operations, including discovery that completes during cancellation.

The server tracks unauthenticated sockets as well as occupied seats. Shutdown
interrupts reads and writes rather than waiting for their normal deadlines or a
TLS close-notify write. A random per-client resumption key allows that same client
to replace its half-open connection. The old reader cannot apply buffered commands
after replacement, and its cleanup cannot clear the replacement seat.

## Recovery guarantees

- Successful authoritative resynchronization resets reconnect backoff. Commands
  are never automatically resent. Match number and expected shot count reject
  obsolete gameplay intents; duplicate accepted shots cannot consume extra turns.
- An interrupted shot may have committed or may not have reached the host.
  The next authoritative view resolves that uncertainty; the client does not guess.
- Router leases request 120 seconds. Renewals happen halfway through the last
  confirmed grant. Transient failures retry at 1, 2, 4, 8, then 10 seconds, bounded
  by confirmed expiry. One dropped response does not delete a usable mapping.
- Expiry withdraws the mapped endpoint and triggers discovery again. Discovery
  failures retry with a capped delay. PCP/NAT-PMP can report a changed assigned
  endpoint; UPnP renewals re-read the WAN address.
- Local interface addresses refresh every ten seconds, including with `--no-map`.
  Valid endpoint changes refresh host invitations and are sent to an authenticated
  guest over the existing pinned connection. Updating routes cannot change the
  host fingerprint, player key, protocol version, or invitation expiry.
- When all paths are already broken, the guest can paste an updated invitation
  with **J**. This updates the existing session rather than creating a new match.
- Expired mappings are never announced as active. Mapping status describes a
  router lease, not proof of public Internet reachability.
- The host never sends an opponent's fleet placements. Player views contain only
  the recipient's fleet and confirmed shot results; unsunk enemy ship identity
  remains concealed.

## Validation evidence

The automated checks exercise behavior rather than only internal field values:

| Area | Evidence |
|---|---|
| Full gameplay | Real pinned-TLS matches through victory, alternating starters and rematch reset; twenty consecutive matches per stress iteration |
| Interrupted commands | Stop reading before a committed shot's reply, reconnect, verify exactly one shot and continue play |
| Delayed commands | Prepare a shot, replace its connection, prove the queued intent is rejected, then accept a fresh intent |
| Reconnect | Repeated forced disconnects, reset backoff, candidate-address fallback, complete route loss repaired with an updated invitation |
| Half-open sessions | Resume a seat without first closing its old socket; old cleanup cannot evict the replacement |
| Slow/malformed peers | Real server read/write loops over a blocked transport with another TLS seat; oversized/truncated requests; bounded decoding |
| Ownership | Same-port replacement, forty rapid starts/stops, concurrent close, parent cancellation, stale UI results, late discovery cleanup |
| Router failure | Simulated-clock dropped replies, expiry and rediscovery after reboot, changed endpoints, short leases only, cancellation, WAN refresh |
| Protocol fuzzing | Invitations, server messages, and commands with atomic-rejection/game-state invariants |
| Presentation | Live two-terminal host/paste/deploy/shoot/disconnect/rejoin/resize/quit; original lookdev PTY checks and renderer matrix |

During this pass, the repeated race-enabled stress gate completed five iterations
(including 100 full matches across five sessions). Each fuzz target subsequently
passed a 100,000-execution bound. One earlier time-bounded message fuzz invocation
ended with a coordinator deadline error and produced no failing input; the
count-bounded rerun passed. `make hardening` uses execution counts for repeatability.

Run:

```sh
make hardening
make test-terminal
GOOS=windows GOARCH=amd64 go build -o /tmp/sea-battle-check.exe .
GOOS=darwin GOARCH=arm64 go build -o /tmp/sea-battle-check-macos .
```

## Limits that remain explicit

Real router firmware, upstream NAT/CGNAT, actual Internet paths, and native
Windows/macOS terminal behavior still need field testing. Simulated router
responses and cross-builds do not prove those environments work.

A private WAN address, IPv6 firewall, or upstream NAT can still block inbound
connections. A new address cannot be delivered automatically after every old
path is gone; an updated invitation must then travel out of band. Failure
detection is bounded by socket/heartbeat deadlines rather than instantaneous.

The host remains trusted and holds both fleets. Match state is in memory; host
exit/restart ends it. Invitations retain their 24-hour lifetime. No save/load,
host migration, relay fallback, or shared rendezvous infrastructure was added.
Router cleanup is best effort, with lease expiry covering crashes or unreachable
routers.
