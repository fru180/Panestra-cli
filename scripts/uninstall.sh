#!/bin/sh
set -eu

if [ -n "${PREFIX:-}" ]; then
  prefix=$PREFIX
elif command -v brew >/dev/null 2>&1; then
  prefix=$(brew --prefix)
else
  prefix=/usr/local
fi
if command -v panestra >/dev/null 2>&1; then
  panestra uninstall
fi
rm -f "$prefix/bin/panestra"
echo "Removed $prefix/bin/panestra"
