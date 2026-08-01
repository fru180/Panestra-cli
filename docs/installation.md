# Installation

## Homebrew

```bash
brew install fru180/tap/panestra-cli
panestra setup
source ~/.zshrc
panestra doctor
```

`setup` は検出できた Codex CLI / Claude Code のみを設定し、複数回実行しても shim、フック、PATH ブロックを重複させません。

Codex のフックは `~/.codex/hooks.json`、Claude Code のフックは `~/.claude/settings.json`（`CLAUDE_CONFIG_DIR` 設定時はその配下）へマージされます。Codex は初回に `/hooks` を開き、追加されたコマンドフックを確認・信頼してください。

## Source install

Go 1.23 以上が必要です。

```bash
./scripts/install.sh
panestra setup
source ~/.zshrc
```

`PREFIX` で配置先を変更できます。

```bash
PREFIX="$HOME/.local" ./scripts/install.sh
```

## Uninstall

```bash
panestra uninstall
```

Homebrew で導入したバイナリも削除する場合:

```bash
brew uninstall panestra-cli
```

Codex CLI、Claude Code、tmux 本体および Panestra CLI 管理外の設定は削除しません。
