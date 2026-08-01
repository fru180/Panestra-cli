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

## Architecture

Panestra CLI consists of one Go binary, `codex` and `claude` shims, and `UserPromptSubmit` hooks for both agents. Interactive launches reuse the current tmux pane or start an isolated tmux server; non-interactive commands execute the agent directly.

Hooks receive JSON on stdin, normalize and truncate the prompt by terminal cell width, and update the pane-local `@panestra_cli_prompt` option. Pass prompt data to tmux through `exec.Command` arguments, never through shell interpolation. Strip ANSI and control sequences without splitting Unicode grapheme clusters.

Setup and uninstall must modify only Panestra CLI-owned hook entries, managed `.zshrc` blocks, and installation files. Existing tmux window settings must be restored after the final Panestra CLI process exits.

## Release Process

- Confirm all development commands above pass, including Terminal.app, iTerm2, Ghostty, Intel Mac, and real-agent smoke tests where available.
- Confirm `HOMEBREW_TAP_TOKEN` has push access and `HOMEBREW_TAP_ENABLED=true` is configured before publishing through the release workflow.
- Merge the reviewed feature branch to `main`, then create and push an annotated version tag such as `v0.2.0`.
- Confirm the workflow publishes arm64 and x86_64 archives plus `checksums.txt`, then updates the Homebrew tap.
- Smoke-test the release with `brew update`, `brew install fru180/tap/panestra-cli`, `panestra setup`, `panestra doctor`, and `panestra uninstall`.
