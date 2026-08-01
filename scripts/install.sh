#!/bin/sh
set -eu

if [ -n "${PREFIX:-}" ]; then
  prefix=$PREFIX
elif command -v brew >/dev/null 2>&1; then
  prefix=$(brew --prefix)
else
  prefix=/usr/local
fi
bindir="$prefix/bin"

if ! command -v go >/dev/null 2>&1; then
  echo "Go 1.23 or newer is required." >&2
  exit 1
fi

mkdir -p "$bindir"
go build -trimpath -ldflags "-s -w" -o "$bindir/panestra" ./cmd/panestra
echo "Installed $bindir/panestra"
echo "Next: panestra setup"
