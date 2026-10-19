# Launch visual review checklist

Persistent review ledger for the launch polish pass. Inventory checked against the current render paths on 2026-10-19. Visual approvals and requested changes are tracked below.

## Review rules

- Status: **Needs review → Changes requested → Ready for review → Approved**.
- Check a box only after the user explicitly approves that item. Implementation or a passing test is not visual approval.
- Record requested changes and evidence in the review log below. Preserve approvals across sessions.
- If later changes affect an approved item, uncheck it and mark **Needs re-review**, with the reason.
- An approval applies only to the variants actually reviewed; record partial coverage in Notes.
- Review SD (80×24) and HD (160×48), both factions where applicable, and multiplayer host/guest differences. Check larger-terminal centering and resizing between sizes.
- For each view: compare title position, margins, spacing, typography, color meaning, panel treatment, control/footer placement, focus/hover/disabled styles, copy tone, and readable status/error text.

## Player-facing views and states

These are review units, not a count of independent renderer functions. Some share a layout but need separate state review.

| Approved | ID | View / state | How to reach | Variants to inspect | Status / notes |
| --- | --- | --- | --- | --- | --- |
| [ ] | V01 | Title / main menu | Launch normally | Idle animation, hover/click reactions, notices, still-motion setting | Ready for review — existing design accepted; added requested bottom-right “by TTY.games” credit in SD/HD. |
| [ ] | V02 | Solo faction selection | Menu → S | United States Navy / Brood, difficulty choices, hover and pressed states | Ready for review — simplified heading and removed supporting subtitles. |
| [ ] | V03 | Field manual | Menu → ? | Rules, controls, multiplayer instructions, back/quit hints | Needs re-review — approved layout retained; multiplayer sentence updated for automatic Join. |
| [ ] | V04 | Settings | Menu → T | Auto/compact display, animated/still menu, three difficulties, save success/failure, C Credits navigation | Ready for review — grouped rows, clickable boolean switches, difficulty selector, Credits row. |
| [ ] | V28 | Credits | Settings → C; Esc returns to Settings | Animated ocean, three contributor credits plus publisher, SD/HD, footer | Ready for review |
| [ ] | V05 | Host: preparing invitation | Menu → H | Helper download/startup, progress, ability to arrange fleet, setup failure/retry | Ready for review — status panel, spaced instructions, clickable fleet/leave footer. |
| [ ] | V06 | Host: invitation ready | Host → I; initially shown when ready | Copy/save controls, feedback/errors, direct/public status, updated invitation | Ready for review — status panel, copy/save buttons, numbered sharing steps, clickable footer. |
| [ ] | V07 | Guest: join invitation | Menu → J | Automatic clipboard read, immediate paste-to-connect, invalid/expired invitation coaching, clickable footer | Ready for review — simplified automatic Join. |
| [ ] | V08 | Connecting / route negotiation | Join → Enter; session startup | Secure connection progress, automatic negotiation, slow/failing setup, cancel | Ready for review — centered status panel, animated dots, brief coaching and clickable Cancel/Quit. |
| [ ] | V09 | Guest: update invitation / rejoin | Disconnected guest → J | Automatic clipboard/paste route update, same-session coaching, invalid replacement, cancel | Ready for review — no Enter required. |
| [ ] | V10 | Fleet deployment | Solo or multiplayer → arrange fleet | Empty/partial/full fleet; all five ships; horizontal/vertical previews; legal/blocked placement; move/remove/undo | Needs review |
| [ ] | V11 | Deployment: submitting / waiting | Deploy → Space | Launching, fleet locked, opponent not ready, disconnected while arranging | Needs review |
| [ ] | V12 | Deployment help overlay | Deployment → ? | Readability over board, complete controls, close affordance | Needs review |
| [ ] | V13 | Battle: targeting enemy waters | Both fleets ready | Your/enemy turn, sending shot, selected cell, unshot/miss/hit/sunk marks, damaged fleet, notices | Needs review |
| [ ] | V14 | Battle: inspect own waters | Battle → Tab / click own chart / fleet name | Swapped chart focus, ship inspection, return to targeting | Needs review |
| [ ] | V15 | Battle help overlay | Battle → ? | Readability over charts, key legend, close affordance; also accessible after match | Needs review |
| [ ] | V16 | Enemy-turn transition | Opponent shot playback | Multiplayer ENEMY TAKING AIM / INCOMING FIRE; solo fade transition | Needs review |
| [ ] | V17 | Miss cinematic | A shot misses | YOU MISSED / THEY MISSED, splash timing, transition/skip | Needs review |
| [ ] | V18 | Hit cinematic | A shot hits without sinking | YOU HIT / YOU WERE HIT, anonymous enemy hit vs known own hull, both factions | Needs review |
| [ ] | V19 | Sinking cinematic | Final hit on a ship | SUNK title and ship name; all ten hulls across both fleets; transition/skip | Needs review |
| [ ] | V20 | Victory result | Win a match | Both factions × close/overwhelming victory; stats, artwork, footer controls | Needs review |
| [ ] | V21 | Defeat result | Lose a match | Both factions × narrow/overwhelming defeat; stats, artwork, footer controls | Needs review |
| [ ] | V22 | Post-match charts | Result → Tab | Final boards, victory/defeat header, fleet condition, return to result | Needs review |
| [ ] | V23 | Rematch waiting | Result / finished charts → R | Waiting for other commander; solo preparing next battle; disabled/requested control | Needs review |
| [ ] | V24 | Leave / quit confirmation | Active session → Q or return to menu | Solo progress warning, host ends-session warning, guest disconnect warning, confirm/cancel | Needs review |
| [ ] | V25 | Connection lost / recovery states | Interrupt an active multiplayer link | Battle LINK LOST / RECONNECTING; deployment waiting; notices; restored session; new invitation path | Needs review |
| [ ] | V26 | Errors and feedback across views | Invalid input, failed startup/action, clipboard/save failures | Menu failure notice; inline action errors; long messages; success feedback; no overlap with controls | Needs review |
| [ ] | V27 | Terminal too small | Resize below 80 columns or 24 rows | Current/required dimensions, resize instructions, recovery to prior view | Needs review |

## Shared visual alignment

- [ ] X01 — Consistent secondary-screen layout: manual, settings, join/rejoin, connecting, invitation, leave confirmation.
- [ ] X02 — Consistent deployment/combat/results headers, margins, key labels, footers, and faction colors.
- [ ] X03 — SD completeness: no truncated instructions, missing actions, overlaps, or offscreen notices.
- [ ] X04 — HD composition and centering; transitions between SD and HD preserve usable layouts.
- [ ] X05 — Mouse hover/pressed/focus feedback agrees with keyboard controls; unavailable actions look unavailable.
- [ ] X06 — Cinematic entry/exit and result transitions feel coherent; text stays legible throughout animation.
- [ ] X07 — Network progress, failures, recovery, and clipboard/save feedback use consistent plain language.

All shared checks start at **Needs review**. Checkboxes require user approval.

## Scripted art review (separate from normal gameplay)

The `--lookdev` mode includes 40 scripted scenes. These are art-review surfaces; they are not 40 additional pages in the normal game. For launch, prioritize the player-facing list above. Scene names below are exact IDs from `internal/visual/catalog.go`.

Run `./bin/sea-battle --lookdev`; Tab opens the scene picker and ? opens lookdev help. Use `--scene ID --paused` to open a particular scene for inspection. Review SD/HD and relevant camera/weather variants.

- [ ] L01 — Lookdev scene-picker overlay (including scrolling and selection).
- [ ] L02 — Lookdev help overlay.
- [ ] L03 — Shared lookdev HUD, playback/pause/slow/tour controls, capture feedback, and small-terminal state.

| Approved | Scene ID | Category | Status / notes |
| --- | --- | --- | --- |
| [ ] | `arrival` | OPENING | Needs review |
| [ ] | `human-station` | COMMAND | Needs review |
| [ ] | `alien-station` | COMMAND | Needs review |
| [ ] | `deployment` | PLACEMENT | Needs review |
| [ ] | `cyclone` | FLEET / A | Needs review |
| [ ] | `virginia` | FLEET / A | Needs review |
| [ ] | `freedom` | FLEET / A | Needs review |
| [ ] | `zumwalt` | FLEET / A | Needs review |
| [ ] | `ford` | FLEET / A | Needs review |
| [ ] | `skiff` | FLEET / B | Needs review |
| [ ] | `lurker` | FLEET / B | Needs review |
| [ ] | `ravager` | FLEET / B | Needs review |
| [ ] | `carapace` | FLEET / B | Needs review |
| [ ] | `hive` | FLEET / B | Needs review |
| [ ] | `human-bow` | CAMERA STUDY | Needs review |
| [ ] | `alien-bow` | CAMERA STUDY | Needs review |
| [ ] | `human-plan` | CAMERA STUDY | Needs review |
| [ ] | `alien-plan` | CAMERA STUDY | Needs review |
| [ ] | `launch` | FIRE CONTROL | Needs review |
| [ ] | `spit` | FIRE CONTROL | Needs review |
| [ ] | `miss` | OUTCOME / MISS | Needs review |
| [ ] | `hit` | OUTCOME / HIT | Needs review |
| [ ] | `retaliation` | OUTCOME / HIT | Needs review |
| [ ] | `sinking` | OUTCOME / SUNK | Needs review |
| [ ] | `loss` | OUTCOME / SUNK | Needs review |
| [ ] | `depth` | UNDERWATER | Needs review |
| [ ] | `storm` | ENVIRONMENT | Needs review |
| [ ] | `victory` | AFTERMATH | Needs review |
| [ ] | `defeat` | AFTERMATH | Needs review |
| [ ] | `bulletin` | TRANSMISSION | Needs review |
| [ ] | `hivecast` | TRANSMISSION | Needs review |
| [ ] | `human-fleet` | FLEET FORMATION | Needs review |
| [ ] | `alien-fleet` | FLEET FORMATION | Needs review |
| [ ] | `exchange` | CINEMATIC SEQUENCE | Needs review |
| [ ] | `counterexchange` | CINEMATIC SEQUENCE | Needs review |
| [ ] | `air-wing` | FLIGHT OPERATIONS | Needs review |
| [ ] | `brood-wing` | BROOD OPERATIONS | Needs review |
| [ ] | `periscope` | OPTICAL CONTACT | Needs review |
| [ ] | `overkill` | CINEMATIC EXCESS | Needs review |
| [ ] | `digestive-event` | CINEMATIC EXCESS | Needs review |

## Review log

| Date | Item | Requested change / evidence | Decision / coverage |
| --- | --- | --- | --- |
| 2026-10-19 | Inventory | Enumerated normal-game render paths, overlays, state variants, and scripted art catalog. | Inventory only; no visual approval. |

| 2026-10-19 | V01 | User: existing menu is great; only requested change is bottom-right “by TTY.games”. Added credit in SD/HD. | Ready for review of credit; no broader redesign requested. |

| 2026-10-19 | V28 / V04 | Added themed Credits page accessible from Settings only, with requested role/name pairs. | Ready for review; Settings navigation updated. |

| 2026-10-19 | V28 | Added Published by / Off Court Creations LLC; adjusted SD/HD spacing for four credits. | Ready for review. |

| 2026-10-19 | V02 | Heading now SINGLE PLAYER; removed Choose your navy line and both supporting faction subtitles. Kept Steel hulls. and Organic nastiness. | Ready for review. |

| 2026-10-19 | V02 / V10 / V13 | Renamed displayed faction to United States Navy in selection, deployment, combat, and art-review caption; updated help and review expectations. | Ready for review of faction label. |

| 2026-10-19 | V03 | Inspected native terminal screenshot; replaced clumped prose with spaced control columns, colored keys, section headings, and a darker ocean backdrop. | Ready for review in SD/HD. |

| 2026-10-19 | V03 | User reviewed the revised Field Manual: “much better”. | Approved; user-reviewed display size unspecified. |

| 2026-10-19 | V04 | Rebuilt Settings with spaced groups, compact/animation switches, selected difficulty, clickable Credits row, and dark ocean backdrop. Inspected SD/HD screenshots; verified mouse toggle and Credits return in a real terminal. | Ready for review. |

| 2026-10-19 | V04 | Confirmed Compact defaults OFF. Reset local saved preference changed by terminal review; isolated terminal-review settings to prevent recurrence. | Ready for review. |

| 2026-10-19 | V03 / V04 / V28 | Unified footer position, labels, divider, branding and hover treatment. Help and Credits now support mouse navigation; Back from Credits returns to Settings. | Footers ready for review; V03 reopened only for footer change. |

| 2026-10-19 | V03 / V04 / V28 | User: “great!” after shared footer update. | Shared footers approved; V03 approval restored. Full V04/V28 approval remains pending. |

| 2026-10-19 | V05 / V06 | Applied shared ocean treatment, spacing and colors; added clickable Copy/Save and fleet/leave actions. Inspected ready-state SD/HD captures; real terminal save, arrange and reopen checks passed. | Ready for review. |

| 2026-10-19 | V05 / V06 | Centered Copy/Save labels horizontally and vertically; centered status text block vertically, including when no subtitle is present. Added three softly pulsing cyan dots during setup, independent of menu-motion preference. Inspected ready/loading SD/HD captures. | Ready for review. |

| 2026-10-19 | V02 / V04 / V06 / V10 / V13 / V23 / X05 | Added shared 240ms press-and-fade feedback for keyboard and mouse controls that remain visible. Covers invitation Copy/Save, settings switches and difficulty, faction difficulty, deployment tools, battle and rematch controls. Feedback follows display resizing and does not leak onto another screen. | Ready for review; actions remain immediate. |

| 2026-10-19 | V06 | Save now atomically replaces invitation.txt and shows its absolute path in a dedicated saved-file panel, wrapping long paths. Private permissions retained; overwrite and symlink replacement tested. | Ready for review. |

| 2026-10-19 | V05 | Removed WE'LL HANDLE THE CONNECTION and its supporting sentence. Moved the loading/status panel toward the center and aligned the fleet/reopen hints beneath it. | Ready for review. |

| 2026-10-19 | V07 / V09 / V06 / V03 | Join reads clipboard only on explicit J/Join; valid clipboard or pasted invitations connect immediately. Added bounded native clipboard helpers and terminal fallback, short coaching, clickable footer, and automatic rejoin input. Updated hosting/manual instructions. Unit and real terminal clipboard/paste connection checks passed; full two-terminal play/reconnect review passed. | Ready for review; V03 connection copy changed and needs re-review. |

| 2026-10-19 | V08 | Replaced old Connecting text block with shared ocean styling, centered status/loading panel and short coaching. Added matching hoverable/clickable Cancel and Quit footer. Inspected SD/HD captures; cancel navigation tested including larger-terminal centering. | Ready for review. |

| 2026-10-19 | V10 / V12 / V03 | Scroll up/down over the deployment board rotates the selected preview, like X/right-click. Updated help/manual controls. Verified both directions, no placement side effect, help overlay and locked fleet guards in SD/HD. | Ready for review. |

## Source map

- `internal/play/play.go`: top-level screen dispatch, connection states, notices, confirmation and terminal-size handling.
- `internal/play/settings.go`: settings and field-manual copy.
- `internal/visual/game.go`: shared front screens, menu, battle dispatch, turn transition and shot cinematics.
- `internal/visual/faction_menu.go`: solo faction selection.
- `internal/visual/placement.go`: deployment and placement-help overlay.
- `internal/visual/combat.go`: battle charts, status, post-match charts and combat-help overlay.
- `internal/visual/result.go`: result variants and rematch waiting.
- `internal/visual/catalog.go`, `main.go`: scripted lookdev scenes and navigation.
