#!/usr/bin/env python3
"""Validate and push staged itch packages to the stable release channels."""
import argparse
import pathlib
import re
import shlex
import subprocess

from stage import TARGETS


def commands(directory: pathlib.Path, target: str, version: str, butler: str) -> list:
    if not re.fullmatch(r"[a-z0-9-]+/[a-z0-9-]+", target):
        raise ValueError("itch target must be account/game")
    version = version.removeprefix("v")
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        raise ValueError("stable itch channels require an X.Y.Z release")
    result = []
    for (system, _), channel in TARGETS.items():
        package = directory / channel
        if (package / "VERSION.txt").read_text().strip() != version:
            raise ValueError(f"wrong package version for {channel}")
        if not (package / ".itch.toml").is_file():
            raise ValueError(f"missing launch manifest for {channel}")
        # Butler's validator only accepts 386/amd64. Unix launchers are scripts;
        # Windows ARM validates with a PE-probe warning and is misreported as
        # amd64. This validates the launch configuration, not ARM compatibility.
        result.append([butler, "validate", "--platform", "osx" if system == "darwin" else system,
                       "--arch", "amd64", str(package)])
    # Validate every channel before making the first public update.
    for channel in TARGETS.values():
        result.append([butler, "push", str(directory / channel), f"{target}:{channel}",
                       "--userversion", version])
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--directory", type=pathlib.Path, default=pathlib.Path("dist/itch"))
    parser.add_argument("--target", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--butler", default="butler")
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()
    for command in commands(args.directory, args.target, args.version, args.butler):
        print(shlex.join(command), flush=True)
        if not args.dry_run:
            subprocess.run(command, check=True)
