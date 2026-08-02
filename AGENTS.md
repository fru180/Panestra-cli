# Repository Guidelines

- Keep `CHANGELOG.md` limited to released versions. Do not add an `Unreleased` section or entries for changes that have not been released.
- Keep development commands out of `README.md`; document them in this file instead.
- If the original plan needs to change, always obtain the user's approval before changing direction.
- When proposing improvements, pursue a fundamental solution instead of an ad hoc minimal fix.
- When implementing tests, cover the necessary cases without omissions or redundancy, and avoid tests that overconstrain implementation details.

## Development Commands

```bash
gofmt -w cmd internal
test -z "$(gofmt -l cmd internal)"
go test ./...
go test -race ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.6.1 ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
go run github.com/zricethezav/gitleaks/v8@v8.30.1 git --redact --no-banner --verbose
scripts/generate-third-party-notices.sh
git diff --exit-code -- THIRD_PARTY_NOTICES
sh -n scripts/*.sh
go build -o /tmp/panestra-cli-check ./cmd/panestra
python3 -m json.tool adapters/codex/hooks/hooks.json >/dev/null
python3 -m json.tool adapters/claude/hooks/hooks.json >/dev/null
```

## Pre-push Verification

- Use the Go toolchain version pinned in `.go-version` for local validation, CI, and release builds. Keep `go.mod` as the minimum source-compatible Go version.
- Before every push, run all commands in `Development Commands`, inspect `git diff --check`, and confirm `git status` contains only the intended changes.
- When deleting or renaming files or directories, search the entire repository for stale references, including `.github/workflows`, before committing.
- Do not rely on CI to discover failures that can be reproduced locally.

## CI Policy

- Pull requests to `stg` run the quality job once on Apple Silicon: formatting, vet, staticcheck, govulncheck, tests, build, shell validation, and adapter JSON validation.
- Pushes to `stg` and `main`, plus pull requests targeting `main`, also run race tests on both Apple Silicon and Intel.
- Keep tmux integration tests path-filtered to tmux, hook, launcher, and asset changes.
- Do not duplicate architecture-independent quality checks across the OS matrix.

## Architecture

Panestra CLI consists of one Go binary, `codex` and `claude` shims, and `UserPromptSubmit` hooks for both agents. Interactive launches reuse the current tmux pane or start an isolated tmux server; non-interactive commands execute the agent directly.

Hooks receive JSON on stdin, normalize and truncate the prompt by terminal cell width, and update the pane-local `@panestra_cli_prompt` option. Pass prompt data to tmux through `exec.Command` arguments, never through shell interpolation. Strip ANSI and control sequences without splitting Unicode grapheme clusters.

Setup and uninstall must modify only Panestra CLI-owned hook entries, managed `.zshrc` blocks, and installation files. Existing tmux window settings must be restored after the final Panestra CLI process exits.

## Repository Privacy

- All commit author/committer metadata and deleted blobs reachable from the public refs as of 2026-08-02 were reviewed. The existing metadata and removed design/demo materials are approved to remain public, so do not rewrite published history solely to remove them.
- Use a GitHub-provided noreply address for future commits. Configure the address for this repository only, and use the noreply address associated with your own GitHub account.
- Run the pinned Gitleaks command in `Development Commands` against full Git history before every push and release. Do not add an allowlist or baseline without documenting why the matched value is safe.

## Release Process

- Confirm all development commands above pass, including Terminal.app, iTerm2, Ghostty, Intel Mac, and real-agent smoke tests where available.
- Confirm `HOMEBREW_TAP_TOKEN` has push access and `HOMEBREW_TAP_ENABLED=true` is configured before publishing through the release workflow.
- Merge the reviewed feature branch to `main`, then create and push an annotated version tag such as `v0.2.0`.
- Confirm the workflow publishes arm64 and x86_64 archives plus `checksums.txt`, then updates the Homebrew tap.
- Smoke-test the release with `brew update`, `brew install fru180/tap/panestra-cli`, `panestra setup`, `panestra doctor`, and `panestra uninstall`.
