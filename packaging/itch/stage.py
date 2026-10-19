#!/usr/bin/env python3
"""Package GoReleaser's exact binaries with itch Play launchers and manifests."""
import argparse
import json
import hashlib
import tarfile
import zipfile
import pathlib
import plistlib
import re
import shutil

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parent.parent
TARGETS = {
    ("linux", "amd64"): "linux-amd64",
    ("linux", "arm64"): "linux-arm64",
    ("darwin", "amd64"): "mac-amd64",
    ("darwin", "arm64"): "mac-arm64",
    ("windows", "amd64"): "windows-amd64",
    ("windows", "arm64"): "windows-arm64",
}


def executable(source: pathlib.Path, destination: pathlib.Path) -> None:
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(source, destination)
    destination.chmod(0o755)


def release_binaries(dist: pathlib.Path, version: str) -> dict:
    """Recover exact executables from retained archives; never rebuild a release."""
    sums = {}
    for line in (dist / "checksums.txt").read_text().splitlines():
        sha, name = line.split()
        if not re.fullmatch(r"[a-f0-9]{64}", sha) or name in sums:
            raise ValueError("invalid checksums")
        sums[name] = sha
    result = {}
    for system, arch in TARGETS:
        entry = "sea-battle.exe" if system == "windows" else "sea-battle"
        extension = "zip" if system == "windows" else "tar.gz"
        archive = dist / f"sea-battle_{version}_{system}_{arch}.{extension}"
        if hashlib.sha256(archive.read_bytes()).hexdigest() != sums.get(archive.name):
            raise ValueError("archive checksum mismatch")
        expected = {entry, "LICENSE", "README.md", "PLAYING.md"}
        if system == "windows":
            with zipfile.ZipFile(archive) as z:
                if len(z.infolist()) != len(expected) or set(z.namelist()) != expected:
                    raise ValueError("unexpected archive members")
                for f in z.infolist():
                    if f.is_dir() or f.file_size > 128 << 20 or (f.external_attr >> 16) & 0o170000 not in (0, 0o100000):
                        raise ValueError("unsafe archive member")
                data = z.read(entry)
        else:
            with tarfile.open(archive, "r:gz") as z:
                members = z.getmembers()
                if len(members) != len(expected) or {f.name for f in members} != expected:
                    raise ValueError("unexpected archive members")
                if any(not f.isfile() or f.size > 128 << 20 for f in members):
                    raise ValueError("unsafe archive member")
                data = z.extractfile(entry).read(128 << 20 | 1)
        target = dist / "recovered" / (system + "-" + arch) / entry
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(data)
        target.chmod(0o755)
        result[(system, arch)] = target
    return result


def stage(dist: pathlib.Path, out: pathlib.Path, release_version=None) -> None:
    metadata = {"version": release_version} if release_version else json.loads((dist / "metadata.json").read_text())
    version = metadata["version"].removeprefix("v")
    if not re.fullmatch(r"\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?", version):
        raise ValueError("invalid release version")
    if release_version:
        binaries = release_binaries(dist, version)
    else:
        binaries = {}
        for artifact in json.loads((dist / "artifacts.json").read_text()):
            if artifact["type"] != "Binary" or artifact.get("extra", {}).get("ID") != "sea-battle":
                continue
            target = (artifact["goos"], artifact["goarch"])
            if target not in TARGETS:
                continue
            if target in binaries:
                raise ValueError(f"duplicate binary for {target}")
            path = pathlib.Path(artifact["path"])
            if not path.is_file():
                raise ValueError(f"missing binary: {path}")
            binaries[target] = path
    if set(binaries) != set(TARGETS):
        raise ValueError(f"missing release targets: {set(TARGETS) - set(binaries)}")
    for target, channel in TARGETS.items():
        directory = out / channel
        if directory.exists():
            shutil.rmtree(directory)
        directory.mkdir(parents=True)
        for name in ["LICENSE", "README.md", "PLAYING.md"]:
            shutil.copyfile(ROOT / name, directory / name)
        shutil.copyfile(HERE / "README-PLAY.txt", directory / "README-PLAY.txt")
        (directory / "VERSION.txt").write_text(version + "\n")
        if target[0] == "linux":
            executable(binaries[target], directory / "sea-battle")
            executable(HERE / "Play.sh", directory / "Play.sh")
            launch_path, platform = "Play.sh", "linux"
        elif target[0] == "windows":
            executable(binaries[target], directory / "sea-battle.exe")
            shutil.copyfile(HERE / "Play.cmd", directory / "Play.cmd")
            launch_path, platform = "sea-battle.exe", "windows"
        else:
            app = directory / "Sea Battle 2.app" / "Contents"
            executable(binaries[target], app / "Resources" / "sea-battle")
            executable(HERE / "app-launch.sh", app / "MacOS" / "Play")
            executable(HERE / "app-play.command", app / "Resources" / "Play.command")
            executable(HERE / "Play.command", directory / "Play.command")
            info = {
                "CFBundleExecutable": "Play",
                "CFBundleIdentifier": "games.tty.seabattle2",
                "CFBundleName": "Sea Battle 2",
                "CFBundleDisplayName": "Sea Battle 2",
                "CFBundlePackageType": "APPL",
                "CFBundleShortVersionString": version.split("-")[0],
                "CFBundleVersion": version.split("-")[0],
                "LSMinimumSystemVersion": "12.0",
                "NSHighResolutionCapable": True,
            }
            (app / "Info.plist").write_bytes(plistlib.dumps(info))
            launch_path, platform = "Sea Battle 2.app", "osx"
        (directory / ".itch.toml").write_text(
            f'[[actions]]\nname = "play"\npath = "{launch_path}"\nplatform = "{platform}"\n'
            + ('console = true\n' if platform == "windows" else '')
        )
        print(f"staged {channel}: {version}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dist", type=pathlib.Path, default=pathlib.Path("dist"))
    parser.add_argument("--out", type=pathlib.Path, default=pathlib.Path("dist/itch"))
    parser.add_argument("--version", help="restage an existing release from verified archives")
    parser.add_argument("--zip", action="store_true", help="also create portable ZIP downloads preserving execute bits")
    args = parser.parse_args()
    stage(args.dist, args.out, args.version)
    if args.zip:
        for channel in TARGETS.values():
            shutil.make_archive(str(args.out / channel), "zip", root_dir=args.out, base_dir=channel)
