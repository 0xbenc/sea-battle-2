Greetings Program!

Sea Battle 2 is a multiplayer recreation of Hasbro’s Battleship board game, played
in the terminal.

- **Multiplayer is directly player-hosted:** Host → copy invitation → Join/paste.
  No accounts, friends system, rendezvous service, relay, or required shared
  infrastructure. Invitations carry connection details rather than referencing
  a centrally stored game code.
- **Minimize network setup, without promising universal connectivity:** automate
  local address discovery and router port mapping where supported; support LAN,
  IPv6, and manual address/port configuration. Explain connection failures and
  offer actionable setup guidance. Some networks will require manual setup or
  swapping which player hosts. Do not depend on public STUN or IP-lookup services.
- **Agreed networking design (not yet implemented):** Go TCP + TLS on one
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
  Below SD, pause play and show resize guidance.
- **Frontload the graphics:** establish compelling art, readable action, and a
  presentation approaching a vertical slice before building the basic simulation.
  This follows the development approach of the sibling `../ascii-bowl` project.
- **Why this order:** Benjamin trusts the AI collaborator to handle familiar game
  simulation requirements; the early uncertainty is the visual direction, where
  his taste and art direction matter most. The presentation establishes the target
  for the underlying systems to grow into.
