#!/usr/bin/env python3
"""Compatibility entry point; normal Host now manages public setup automatically."""
import os
import pathlib
import sys

game = pathlib.Path(__file__).resolve().parents[1] / "bin" / "sea-battle"
if not game.is_file():
    sys.exit("Build the game first: make build")
os.execv(str(game), [str(game), "--host", *sys.argv[1:]])
