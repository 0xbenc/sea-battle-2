# Playing Sea Battle 2

Run `sea-battle` to open the game. Choose **Single player**, or **Host a battle**
to play with a friend. A terminal at least 80 columns by 24 rows is required.

For multiplayer, copy the host's invitation and send it privately to your friend.
They choose **Join**, then paste it if their clipboard wasn't read automatically.
Both games must stay running. Players can be on different networks. Public
connection setup downloads a checksum-verified cloudflared helper on demand;
solo play does not need it. Internet access is required for this public route.

Place ships with the mouse or arrow keys. Wheel/X rotates, Click/Enter places,
R shuffles and Space launches. In battle, Click/Enter fires, Tab switches charts,
and ? opens help. Each player gets one shot per turn, including hits.

Settings persist in the operating system's user configuration directory under
`sea-battle-2/settings.json`. Matches are not saved across quitting the host.
Saving an invitation writes `invitation.txt` to the current working directory;
the game shows its full path. Share invitations privately.

## Downloaded packages

- Linux: open **Play.sh**, or run `./sea-battle` in a terminal.
- macOS: open **Sea Battle 2.app** or **Play.command**. These open Terminal.
- Windows: itch Play opens a console. For a browser download, use **Play.cmd**.

Launchers do not install anything or change PATH. Keep the extracted package
intact. If a launcher cannot open a terminal, start the executable from an existing
terminal. macOS packages are currently unsigned and not notarized.

## Versions and updates

Open **Settings** to see your version, or run `sea-battle --version`. Local
`make run` builds show the Git revision and whether there are uncommitted changes.

Close the game before updating; an active match cannot survive a host restart.
Settings stay in your user config directory when you replace the game package.

- **Homebrew:** run `brew update`, then `brew upgrade 0xbenc/tap/sea-battle`.
- **itch app:** use the app's update controls for Sea Battle 2.
- **Browser download:** extract the newer package into a new folder and launch it.
  Keep the old folder until you've checked the new version. Saved invitations
  in that folder are private and only useful while their host session is alive.

TTY.games installation is being prepared and is not available yet. Sea Battle
does not check for updates or download a replacement game while you're playing.
