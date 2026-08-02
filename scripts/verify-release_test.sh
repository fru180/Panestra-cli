#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
verifier="$script_dir/verify-release.sh"
test_dir=$(mktemp -d "${TMPDIR:-/tmp}/panestra-release-test.XXXXXX")
trap 'rm -rf "$test_dir"' EXIT HUP INT TERM

repo="$test_dir/repo"
git init -q -b main "$repo"
git -C "$repo" config user.name "Panestra Release Test"
git -C "$repo" config user.email "release-test@example.invalid"
git -C "$repo" commit -q --allow-empty -m "main"

binary="$test_dir/panestra"
printf '%s\n' '#!/bin/sh' 'printf "panestra %s\n" "${FAKE_VERSION:-0.1.0}"' > "$binary"
chmod +x "$binary"

git -C "$repo" tag -a v0.1.0 -m "v0.1.0"
(cd "$repo" && "$verifier" v0.1.0 main "$binary") >/dev/null

expect_failure() {
  description=$1
  shift
  if (cd "$repo" && "$@") >/dev/null 2>&1; then
    echo "Expected failure: $description" >&2
    exit 1
  fi
}

git -C "$repo" tag v0.1.1
expect_failure "lightweight tag" env FAKE_VERSION=0.1.1 "$verifier" v0.1.1 main "$binary"

git -C "$repo" tag -a release-0.1.0 -m "invalid tag"
expect_failure "invalid tag name" "$verifier" release-0.1.0 main "$binary"

git -C "$repo" tag -a v0.1.2 -m "mismatched version"
expect_failure "binary version mismatch" "$verifier" v0.1.2 main "$binary"

git -C "$repo" switch -q --orphan off-main
git -C "$repo" commit -q --allow-empty -m "off main"
git -C "$repo" tag -a v0.1.3 -m "off-main release"
git -C "$repo" switch -q main
expect_failure "tag outside main" env FAKE_VERSION=0.1.3 "$verifier" v0.1.3 main "$binary"

echo "Release verification tests passed"
