# Release Operations

The repository owner, `fru180`, is responsible for release approval and for the `HOMEBREW_TAP_TOKEN` lifecycle. Never place token values in source files, issues, pull requests, workflow logs, or chat messages.

## Homebrew credentials

Create a fine-grained personal access token with these limits:

- Resource owner: `fru180`
- Repository access: only `fru180/homebrew-tap`
- Repository permission: Contents, read and write
- Expiration: 90 days

Store it as the `HOMEBREW_TAP_TOKEN` Actions secret in `fru180/Panestra-cli`, and set the Actions variable `HOMEBREW_TAP_ENABLED` to `true`. Record the expiration in the maintainer's private calendar. Rotate the token before expiration by creating a replacement with the same scope, updating the Actions secret directly in GitHub, verifying repository access, and revoking the old token. No release tag should be pushed while the secret is missing or expired.

## Publishing

Run every command in `AGENTS.md` and complete the available real-terminal and real-agent smoke tests. Merge the reviewed release commit to `main`, then create and push an annotated tag:

```sh
git switch main
git pull --ff-only origin main
git tag -a v0.1.0 -m "Panestra CLI v0.1.0"
git push origin v0.1.0
```

The release workflow accepts only an annotated `vX.Y.Z` tag whose commit is reachable from `origin/main` and whose version matches `panestra version`. A manual workflow run builds and validates artifacts but never publishes them.

## Verification and recovery

Confirm that the release contains both macOS archives, `checksums.txt`, and both SPDX JSON SBOMs. Verify the downloads and attestations:

```sh
shasum -a 256 -c checksums.txt
gh attestation verify panestra-cli_Darwin_arm64.tar.gz --repo fru180/Panestra-cli
gh attestation verify panestra-cli_Darwin_x86_64.tar.gz --repo fru180/Panestra-cli
```

Then run `brew update`, install `fru180/tap/panestra-cli`, and exercise `panestra setup`, `panestra doctor`, and `panestra uninstall` in a clean environment.

If the workflow fails before the publish job, fix the cause and create a new commit and tag; do not move or reuse a published tag. If GitHub Release succeeds but the Homebrew job fails, leave the immutable release and tag in place, correct the Tap credential or workflow failure, and rerun only the failed Homebrew job. Never force-push `main` or a release tag.
