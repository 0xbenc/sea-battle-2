#!/usr/bin/env bash
set -euo pipefail

# CI uses the same tested Linux/amd64 Butler version as local release checks.
destination=${1:?usage: install-butler.sh DESTINATION}
version=15.31.0
checksum=4f2a3f22b12f870923504d4b6935535cad377b45859f5fe9419e3adc0611a48c
mkdir -p "$destination"
curl --fail --location --retry 3 --silent --show-error \
    "https://broth.itch.zone/butler/linux-amd64/$version/archive/default" \
    -o "$destination/butler.zip"
printf '%s  %s\n' "$checksum" "$destination/butler.zip" | sha256sum --check --status
unzip -o -q "$destination/butler.zip" -d "$destination"
chmod +x "$destination/butler"
"$destination/butler" version
