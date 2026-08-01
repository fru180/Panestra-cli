# Release v0.1.0

## Prerequisites

- Create the public `fru180/homebrew-tap` repository.
- Add repository secret `HOMEBREW_TAP_TOKEN` with push access to the Tap.
- Add repository variable `HOMEBREW_TAP_ENABLED=true`.
- Confirm the local validation commands in `requirements-audit.md` pass.
- Complete Terminal.app, iTerm2, Ghostty, Intel Mac, and real Claude Code smoke tests.

## Publish source

Commit the reviewed work on a feature branch, push it, and merge it to `main`. Confirm `test` and `tmux-integration` are green.

## Release

Create and push the version tag:

```bash
git tag -a v0.1.0 -m "Panestra CLI v0.1.0"
git push origin v0.1.0
```

The release workflow builds:

```text
panestra-cli_Darwin_arm64.tar.gz
panestra-cli_Darwin_x86_64.tar.gz
checksums.txt
```

It then creates the GitHub Release and, when the Tap variable is enabled, generates and pushes `Formula/panestra-cli.rb`.

## Repository metadata

Set these topics on `fru180/Panestra-cli`:

```text
panestra
ai-coding-agent
codex-cli
claude-code
developer-tools
tmux
terminal
cli
macos
```

Add this link to the Panestra Web README:

```markdown
For a lightweight terminal-only experience without the web dashboard,
see [Panestra CLI](https://github.com/fru180/Panestra-cli).
```

## Homebrew smoke test

```bash
brew update
brew install fru180/tap/panestra-cli
panestra version
panestra setup
source ~/.zshrc
panestra doctor
```

Verify both release archives against `checksums.txt` and confirm that uninstall restores the pre-install configuration.
