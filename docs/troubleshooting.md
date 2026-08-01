# Troubleshooting

最初に診断を実行します。

```bash
panestra doctor
```

## shim が有効にならない

```bash
source ~/.zshrc
command -v codex
command -v claude
```

パスは `~/.local/share/panestra-cli/shims/` を指す必要があります。別の PATH 設定が後から先頭へ追加している場合は、`.zshrc` の Panestra CLI 管理ブロックをその設定より後に置いてください。

## Codex で表示が更新されない

Codex で `/hooks` を開き、`UserPromptSubmit` の Panestra CLI フックを信頼してください。`[features] hooks = false` が設定されていないことも確認します。

## Claude Code で表示が更新されない

Claude Code で `/hooks` を開き、User scope の `UserPromptSubmit` を確認してください。`~/.claude/settings.json` の `disableAllHooks` が `true` の場合はフックが実行されません。

## 詳細ログ

```bash
PANESTRA_CLI_DEBUG=1 codex
```

フックは通常何も出力せず、JSON や tmux 操作に失敗してもプロンプト送信を止めません。デバッグ時のみ stderr に補助エラーを表示します。

## 実行中・終了後の出力をスクロールする

通常のTerminalから起動した場合も、マウスホイールまたはトラックパッドで過去の出力をスクロールできます。実行中はPanestra CLI専用tmuxのcopy-modeへ自動的に入り、最下部まで戻ると最新出力の追従へ復帰します。CodexまたはClaude Codeの終了後は、プロンプト枠を除いたpane履歴が通常のTerminalへ引き継がれます。

## 一時的に迂回する

```bash
PANESTRA_CLI_DISABLE=1 codex
panestra disable
panestra enable
```
