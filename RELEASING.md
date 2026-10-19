# Sea Battle 2 releases

This first release stage supports **itch.io and Homebrew only**. TTY.games
publishing, installer registration and AWS changes are deferred until those two
routes have been tested. No TTY.games publishing workflow or credentials are used.

The player command is `sea-battle`. The GitHub repository is
`0xbenc/sea-battle-2`; release archive names begin with `sea-battle_`.

## Test before tagging

A push to main or pull request runs tests/vet on Linux, macOS and Windows, then
builds six snapshot binaries: amd64 and arm64 for each OS. CI attaches portable
itch test packages, including terminal-opening launchers. These are test downloads,
not public itch or Homebrew publications. Snapshot versions identify the source
commit; stable versions come from the release tag.

Locally, with Go 1.26.3+, GoReleaser v2.17.0 and Python 3:

```sh
make test
make test-packaging
goreleaser check
goreleaser release --snapshot --clean --skip=sbom
python3 packaging/itch/stage.py --zip
```

The art reader is included under `internal/art`, pinned to the source revision in
`UPSTREAM.txt`. There is no sibling dotmason checkout or private module access.

Test the extracted packages, including paths with spaces, resizing, mouse input,
solo play, Host/Join over different networks, exiting and settings preservation.
Also test macOS app launch and Windows itch Play/native console. Cross-compilation
and package validation do not establish native platform acceptance.

Windows ARM builds can play solo and use direct routes, but the current automatic
public helper download does not support Windows ARM. An explicitly installed
compatible cloudflared may be discovered; that path is not yet verified. Do not
advertise automatic public hosting on Windows ARM as tested.
macOS builds are unsigned/not notarized. Butler 15.31.0 validates ARM launch
manifests using its amd64 option because its validator does not recognize ARM;
this is not architecture validation.

## Destination setup

Create/confirm the itch project and configure these **Sea Battle repository**
Actions settings. Existing termtd repository secrets are not inherited:

| Setting | Purpose |
| --- | --- |
| Secret `TAP_GITHUB_TOKEN` | Contents write access to `0xbenc/homebrew-tap` |
| Secret `BUTLER_API_KEY` | Butler credential authorized for the itch project |
| Variable `ITCH_TARGET` | Confirmed target `tty-games/sea-battle-2` |

Use GitHub's encrypted secret settings or `gh secret set` with a private prompt or
credential pipe. Never commit tokens. A logged-in local gh/Butler installation can
provision these without displaying the values. `GITHUB_TOKEN` for release assets
is provided by Actions. Each destination checks its own credentials; an absent
secret fails that job rather than preventing the GitHub build or the other job.

## Stable publication

After package acceptance, choose an unused stable `vX.Y.Z` tag and push it.
There is no release tag created by initial setup. Stable tags trigger:

1. Native tests/vet and packaging tests on Linux, macOS and Windows.
2. One GoReleaser build, pinned to the triggering tag; six archives, checksums,
   SBOMs and GitHub provenance attestations.
3. Independent Homebrew and itch jobs, each fetching those retained archives and
   verifying both GitHub asset digests and `checksums.txt` before use.

Homebrew generates `Formula/sea-battle.rb` from the real archive checksums. Its
first publication creates the formula; retries are idempotent and older versions
are rejected. The formula supports macOS/Linux amd64/arm64:

```sh
brew install 0xbenc/tap/sea-battle
sea-battle --version
brew test 0xbenc/tap/sea-battle
```

itch stages the exact released executables with Play launchers, validates all six
packages, then uploads `linux-amd64`, `linux-arm64`, `mac-amd64`, `mac-arm64`,
`windows-amd64` and `windows-arm64` with the same user version. Channel uploads are
sequential, not atomic. `butler status ACCOUNT/PROJECT` checks uploaded versions.

Prerelease tags build GitHub release assets but do not update either stable
destination. They do not publish to TTY.games.

## Retry without rebuilding

Actions → **Publish existing release** → Run workflow from **main**. Supply the
retained release tag and destination `homebrew`, `itch` or `all`. Homebrew rejects
downgrades; itch requires the latest stable GitHub release. Never move a published
tag or rebuild its archives to recover a destination failure.

## Player state

Settings live in the OS configuration directory under `sea-battle-2/settings.json`,
outside the package. Public-connection helpers are cached separately in the OS
cache directory. Host sessions are in memory, and quitting the host ends the match.
Invitation files are explicitly saved to the working directory and are private
connection credentials; never include them in release archives.

## Setup verification (2026-10-19)

- All six snapshot archives built; GoReleaser configuration and workflow syntax passed.
- Every itch package preserved its exact release executable and passed Butler's
  launch-manifest validation (ARM limitations above still apply).
- The Linux package reached the menu, solo deployment and battle in a real PTY,
  then restored terminal settings on exit.
- A temporary local Homebrew tap installed the snapshot archive and passed
  `brew test`; that installation and temporary tap were removed afterward.
- Owner confirmed `tty-games/sea-battle-2`. After local GitHub and Butler sign-in,
  `TAP_GITHUB_TOKEN`, `BUTLER_API_KEY` and `ITCH_TARGET` were configured in the
  Sea Battle repository; secret names and timestamps were verified without
  displaying values. GitHub repository administration and tap write access passed.
- Butler status successfully read termtd's six existing channels; Sea Battle
  currently reports no uploaded channels. Actual Sea Battle upload authorization
  remains to be exercised by its first publication.
- CI run 39323226376 passed Linux/macOS/Windows tests and uploaded all six test
  packages. Native Windows Go tests/vet and the Windows Play.cmd version smoke
  test also passed locally after making Unix file-mode assertions platform-aware.
- No stable tag or public itch/tap release has been created by this setup.

## First stable release approval

The owner tested the installed Linux/WSL and native Windows packages and reported
“they run great!”, then approved publication to itch and Homebrew. The first
stable version is **1.0.0**. TTY.games integration remains deferred. The earlier
no-public-release statements above describe the setup checkpoint.
