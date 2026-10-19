#!/usr/bin/env python3
"""Generate the tap formula from a stable release's actual archive checksums.

Usage: render.py VERSION CHECKSUMS_FILE OUT_FILE (accepts an optional leading v).
"""
import pathlib
import re
import sys

ARCHES = {
    "darwin_arm64": "SHA_DARWIN_ARM64",
    "darwin_amd64": "SHA_DARWIN_AMD64",
    "linux_arm64": "SHA_LINUX_ARM64",
    "linux_amd64": "SHA_LINUX_AMD64",
}


def render(version: str, checksums: str) -> str:
    version = version.removeprefix("v")
    if not re.fullmatch(r"(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)", version):
        raise ValueError("the tap requires a stable X.Y.Z release")
    sums = {}
    for line in checksums.splitlines():
        if not line.strip():
            continue
        digest, name = line.split()
        if not re.fullmatch(r"[0-9a-fA-F]{64}", digest):
            raise ValueError(f"invalid SHA-256 for {name}")
        if name in sums:
            raise ValueError(f"duplicate checksum for {name}")
        sums[name] = digest.lower()
    formula = pathlib.Path(__file__).with_name("sea-battle.rb.tmpl").read_text()
    formula = formula.replace("{{VERSION}}", version)
    for arch, token in ARCHES.items():
        archive = f"sea-battle_{version}_{arch}.tar.gz"
        if archive not in sums:
            raise ValueError(f"missing {archive} in checksums")
        formula = formula.replace("{{" + token + "}}", sums[archive])
    if "{{" in formula:
        raise ValueError("unsubstituted placeholder in formula")
    return formula


def main() -> int:
    if len(sys.argv) != 4:
        print(__doc__, file=sys.stderr)
        return 2
    try:
        formula = render(sys.argv[1], pathlib.Path(sys.argv[2]).read_text())
        out = pathlib.Path(sys.argv[3])
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(formula)
    except (ValueError, OSError) as error:
        print(error, file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
