# Panestra CLI

[English](#english) | [日本語](#日本語)

## English

Panestra CLI keeps the latest prompt sent to Codex CLI or Claude Code visible at the top of your terminal.

```text
┌ Task: Refactor the authentication flow ─────────────────────────────┐
│ Codex / Claude Code                                                 │
│ Agent responses and execution logs                                 │
```

### Requirements

- macOS (Apple Silicon / Intel)
- zsh
- tmux
- Codex CLI or Claude Code

### Installation

Use Homebrew for the official release:

```bash
brew install fru180/tap/panestra-cli
panestra setup
source ~/.zshrc
panestra doctor
```

To try Panestra CLI from source:

```bash
git clone https://github.com/fru180/Panestra-cli.git
cd Panestra-cli
./scripts/install.sh
panestra setup
source ~/.zshrc
```

Source installation requires Go 1.23 or later. Set `PREFIX` to change the installation destination, for example `PREFIX="$HOME/.local" ./scripts/install.sh`.

`panestra setup` configures only the installed agents and can be run repeatedly without duplicating shims, hooks, or PATH entries. For Codex CLI, open `/hooks` once after setup and trust the added command hook.

After setup, launch your agent as usual:

```bash
codex
claude
```

Only interactive sessions are automatically launched inside tmux. Non-interactive usage, including `codex exec`, `codex review`, `claude -p`, `--help`, `--version`, and piped input, runs unchanged.

### Commands

```text
panestra setup       Install shims, PATH configuration, and agent hooks
panestra doctor      Diagnose the installation
panestra enable      Enable Panestra CLI
panestra disable     Keep Panestra CLI disabled
panestra clear       Clear the display in the current pane
panestra uninstall   Remove configuration added by Panestra CLI
panestra version     Show the version
```

To disable Panestra CLI for a single command, run `PANESTRA_CLI_DISABLE=1 codex`.

To remove Panestra CLI, run `panestra uninstall`. If it was installed with Homebrew, also run `brew uninstall panestra-cli`. Agent installations, tmux, and configuration not managed by Panestra CLI are preserved.

Configuration is stored in `~/.config/panestra-cli/config.toml`:

```toml
enabled = true
auto_tmux = true
prefix = "Task: "
max_width = 120
show_waiting_message = true
```

Environment variables `PANESTRA_CLI_DISABLE`, `PANESTRA_CLI_MAX_WIDTH`, `PANESTRA_CLI_PREFIX`, and `PANESTRA_CLI_DEBUG` override the corresponding settings.

### Troubleshooting

Start with `panestra doctor`. If an agent shim is not active, reload zsh and confirm that `command -v codex` or `command -v claude` points into `~/.local/share/panestra-cli/shims/`.

If prompts do not update, inspect `/hooks` in the affected agent. Codex hooks must be trusted and enabled; Claude Code must have an enabled user-scope `UserPromptSubmit` hook. Run with `PANESTRA_CLI_DEBUG=1` to print diagnostic errors to stderr.

In dedicated tmux sessions, use the mouse wheel or trackpad to browse output. Scrolling back to the bottom resumes live output.

### Privacy

Panestra CLI does not save prompts to files or perform external communication or telemetry. The latest prompt is held temporarily only in a tmux pane option and is cleared when the session ends.

## 日本語

Panestra CLIは、Codex CLIやClaude Codeへ送信した最新の指示をターミナル上部へ固定表示します。

```text
┌ Task: 認証処理をリファクタリングしてください ──────────────────────┐
│ Codex / Claude Code                                                 │
│ エージェントの応答や実行ログ                                       │
```

### 必要環境

- macOS（Apple Silicon / Intel）
- zsh
- tmux
- Codex CLI または Claude Code

### インストール

正式リリースでは Homebrew を使用します。

```bash
brew install fru180/tap/panestra-cli
panestra setup
source ~/.zshrc
panestra doctor
```

ソースから試す場合:

```bash
git clone https://github.com/fru180/Panestra-cli.git
cd Panestra-cli
./scripts/install.sh
panestra setup
source ~/.zshrc
```

ソースからのインストールには Go 1.23 以降が必要です。インストール先を変更する場合は、`PREFIX="$HOME/.local" ./scripts/install.sh` のように `PREFIX` を指定します。

`panestra setup` はインストール済みのエージェントだけを設定し、複数回実行しても shim、フック、PATH 設定を重複させません。Codex CLIでは、セットアップ後に一度 `/hooks` を開き、追加されたコマンドフックを信頼してください。

セットアップ後も、通常どおり起動します。

```bash
codex
claude
```

対話セッションだけが自動的に tmux 内で起動します。`codex exec`、`codex review`、`claude -p`、`--help`、`--version`、パイプ入力などの非対話利用はそのまま実行されます。

### コマンド

```text
panestra setup       shim・PATH・エージェントフックを導入
panestra doctor      導入状態を診断
panestra enable      有効化
panestra disable     継続的に無効化
panestra clear       現在のペインの表示を消去
panestra uninstall   Panestra CLI が追加した設定を削除
panestra version     バージョンを表示
```

1回だけ無効化するには `PANESTRA_CLI_DISABLE=1 codex` を使用します。

Panestra CLIを削除するには `panestra uninstall` を実行します。Homebrewでインストールした場合は、続けて `brew uninstall panestra-cli` を実行します。エージェント本体、tmux、Panestra CLIの管理外にある設定は削除されません。

設定は `~/.config/panestra-cli/config.toml` にあります。

```toml
enabled = true
auto_tmux = true
prefix = "Task: "
max_width = 120
show_waiting_message = true
```

環境変数 `PANESTRA_CLI_DISABLE`、`PANESTRA_CLI_MAX_WIDTH`、`PANESTRA_CLI_PREFIX`、`PANESTRA_CLI_DEBUG` で上書きできます。

### トラブルシューティング

最初に `panestra doctor` を実行してください。エージェントの shim が有効にならない場合はzshを再読み込みし、`command -v codex` または `command -v claude` が `~/.local/share/panestra-cli/shims/` 配下を指していることを確認します。

プロンプト表示が更新されない場合は、対象エージェントの `/hooks` を確認します。Codexのフックは信頼済みかつ有効である必要があり、Claude CodeではUser scopeの `UserPromptSubmit` フックが有効である必要があります。`PANESTRA_CLI_DEBUG=1` を指定して実行すると、診断用エラーがstderrへ出力されます。

専用tmuxセッションでは、マウスホイールまたはトラックパッドで過去の出力を確認できます。最下部まで戻ると最新出力の追従を再開します。

### プライバシー

Panestra CLI はプロンプトをファイルへ保存せず、外部通信やテレメトリーを行いません。最新プロンプトは tmux のペインオプションにだけ一時保持され、セッション終了時に消去されます。
