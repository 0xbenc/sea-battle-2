#!/usr/bin/env python3
"""Prevent replay of an older release from downgrading the shared tap."""
import re
import sys
from pathlib import Path

def version(value):
    if not re.fullmatch(r'v?\d+\.\d+\.\d+', value):
        raise ValueError('stable semantic version required')
    return tuple(map(int, value.removeprefix('v').split('.')))

if __name__ == '__main__':
    requested = version(sys.argv[1])
    if not Path(sys.argv[2]).exists():
        raise SystemExit(0)  # First publication of this formula.
    existing = re.search(r'^\s*version "([^"]+)"', Path(sys.argv[2]).read_text(), re.MULTILINE)
    if not existing:
        raise SystemExit('Cannot determine current tap version; refusing overwrite')
    if requested < version(existing[1]):
        raise SystemExit('Refusing to downgrade the tap; select the current or a newer release')
