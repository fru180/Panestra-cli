#!/bin/sh
set -eu

if [ "$#" -ne 3 ]; then
  echo "Usage: $0 <tag> <main-ref> <panestra-binary>" >&2
  exit 2
fi

tag=$1
main_ref=$2
binary=$3

if ! printf '%s\n' "$tag" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'; then
  echo "Release tag must use the vX.Y.Z format: $tag" >&2
  exit 1
fi

tag_ref="refs/tags/$tag"
if [ "$(git cat-file -t "$tag_ref" 2>/dev/null || true)" != "tag" ]; then
  echo "Release tag must be annotated: $tag" >&2
  exit 1
fi

release_commit=$(git rev-parse "$tag_ref^{commit}")
main_commit=$(git rev-parse "$main_ref^{commit}")
if ! git merge-base --is-ancestor "$release_commit" "$main_commit"; then
  echo "Release tag $tag does not point to a commit reachable from $main_ref" >&2
  exit 1
fi

if [ ! -x "$binary" ]; then
  echo "Release binary is not executable: $binary" >&2
  exit 1
fi

expected_version="panestra ${tag#v}"
actual_version=$($binary version)
if [ "$actual_version" != "$expected_version" ]; then
  echo "Version mismatch: tag expects '$expected_version', binary reports '$actual_version'" >&2
  exit 1
fi

printf 'Verified %s at %s with %s\n' "$tag" "$release_commit" "$actual_version"
