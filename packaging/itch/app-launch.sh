#!/bin/sh
set -eu

launcher_dir=$(CDPATH='' cd "$(dirname "$0")" && pwd)
exec /usr/bin/open -a Terminal "$launcher_dir/../Resources/Play.command"
