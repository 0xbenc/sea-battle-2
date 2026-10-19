#!/usr/bin/env bash
set -euo pipefail

game_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
game="$game_dir/sea-battle"

# Running from an existing terminal needs no additional window.
if [[ -t 0 && -t 1 ]]; then
    exec "$game" "$@"
fi

# Use the desktop's preferred terminal when it provides this interface.
if command -v xdg-terminal-exec >/dev/null 2>&1; then
    exec xdg-terminal-exec "$game" "$@"
fi

# Pass paths and game arguments as separate arguments, including spaces.
for terminal in gnome-terminal kgx konsole kitty alacritty wezterm foot x-terminal-emulator xterm; do
    if ! command -v "$terminal" >/dev/null 2>&1; then
        continue
    fi
    case "$terminal" in
        gnome-terminal|kgx) exec "$terminal" -- "$game" "$@" ;;
        kitty|foot) exec "$terminal" "$game" "$@" ;;
        wezterm) exec "$terminal" start -- "$game" "$@" ;;
        *) exec "$terminal" -e "$game" "$@" ;;
    esac
done

# Xfce's launcher accepts a command string, so quote every argument for bash.
if command -v xfce4-terminal >/dev/null 2>&1; then
    printf -v command_line '%q ' "$game" "$@"
    exec xfce4-terminal --disable-server --command "bash -c $(printf '%q' "exec $command_line")"
fi

message="Sea Battle 2 needs a terminal. Open a terminal in this folder and run ./sea-battle, or install a terminal emulator and try Play again."
printf '%s\n' "$message" >&2
if command -v zenity >/dev/null 2>&1; then
    zenity --error --title="Sea Battle 2" --text="$message" || true
fi
exit 1
