#!/usr/bin/env python3
"""Host through an optional public Cloudflare Quick Tunnel; Ctrl-C closes both."""
import os
import pathlib
import re
import signal
import socket
import subprocess
import sys
import tempfile
import time

ROOT = pathlib.Path(__file__).resolve().parents[1]


def main():
    cloudflared = os.environ.get("CLOUDFLARED", str(ROOT / "bin/cloudflared"))
    game = ROOT / "bin/sea-battle"
    if not game.is_file():
        raise RuntimeError("Build the game first: make build")
    with socket.socket() as listener:
        listener.bind(("127.0.0.1", 0))
        port = listener.getsockname()[1]
    # Keep all temporary files inside the game checkout. No system service/config.
    with tempfile.TemporaryDirectory(prefix="public-host-", dir=ROOT / "bin") as directory:
        log_path = pathlib.Path(directory) / "tunnel.log"
        with log_path.open("w") as log:
            tunnel = subprocess.Popen(
                [cloudflared, "tunnel", "--url", f"http://127.0.0.1:{port}", "--no-autoupdate"],
                stdout=log, stderr=subprocess.STDOUT, start_new_session=True,
            )
            try:
                print("Opening a temporary public tunnel…", flush=True)
                deadline = time.monotonic() + 45
                while True:
                    output = log_path.read_text()
                    url = re.search(r"https://[a-z0-9-]+\.trycloudflare\.com", output)
                    if url and "Registered tunnel connection" in output:
                        break
                    if tunnel.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError("Public tunnel failed to start:\n" + output[-4000:])
                    time.sleep(0.2)
                address = url.group().replace("https://", "wss://") + "/sb2"
                print("Public route ready. Copy the game's full invitation to your guest.", flush=True)
                return subprocess.call(
                    [str(game), "--host", "--no-map", "--websocket-port", str(port),
                     "--advertise", address, *sys.argv[1:]], cwd=ROOT,
                )
            finally:
                if tunnel.poll() is None:
                    tunnel.send_signal(signal.SIGTERM)
                    try:
                        tunnel.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        tunnel.kill()
                        tunnel.wait()


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        sys.exit(130)
    except (OSError, RuntimeError) as error:
        print(error, file=sys.stderr)
        sys.exit(1)
