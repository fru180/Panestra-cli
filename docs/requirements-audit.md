# v0.1.0 Requirements Audit

`docs/design.md` の完了条件を、ローカル実装で証明できる項目と外部公開が必要な項目に分けて追跡する。

## Implemented and verified locally

| Area | Requirement | Evidence |
| --- | --- | --- |
| Launch | `codex` / `claude` shim、tmux外の専用サーバー、tmux内の再利用 | `internal/launcher`, dedicated-tmux PTY E2E |
| Launch | 非対話コマンドを迂回し、引数・環境・終了コードを保持 | `interactive_test.go`, isolated lifecycle E2E |
| Display | Waiting表示、最新プロンプトへの更新、`clear` | `internal/tmux`, `handler_integration_test.go` |
| Display | 日本語・絵文字・結合文字・ANSI・制御文字・長文省略 | `prompt_test.go` |
| Safety | `TMUX_PANE`で対象を限定し、他ペインへ干渉しない | `handler_integration_test.go` |
| Safety | シェルを経由せず特殊文字をtmuxへ渡す | `client_integration_test.go` |
| Safety | 既存tmux設定の保存、参照カウント、復元 | `client_integration_test.go` |
| Safety | 不正JSONやtmux障害でフックを失敗させない | `event_test.go`, `TestHandleFailsOpen` |
| Privacy | プロンプトのファイル保存、履歴、ネットワーク処理なし | `internal/hook`, source audit |
| Setup | macOS/zsh/tmux/agent確認、shim、PATH、設定、adapter、doctor | `internal/installer` |
| Setup | Codexのみ、Claudeのみ、両方、2回実行の冪等性 | `TestSetupAgentCombinations` |
| Setup | 既存JSON・`.zshrc`・パーミッションを維持 | `installer_test.go` |
| Management | doctor、enable、disable、clear、uninstall | `cmd/panestra`, isolated lifecycle E2E |
| Adapters | 現行Codex / Claude Code `UserPromptSubmit` JSON | `adapters`, `event_test.go`, official docs review |
| Distribution | README、デモGIF、MIT License、Security、Contributing、Changelog | repository artifacts |
| Automation | macOS Intel/Apple Silicon test、integration、release archives、checksums、Tap update | `.github/workflows` |

Verification command:

```bash
gofmt -w cmd internal scripts/generate-demo
go test ./...
go test -race ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.6.1 ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
sh -n scripts/*.sh
go build ./cmd/panestra
```

## External publication state

Read-only audit on 2026-08-01:

| Requirement | Current evidence | Status |
| --- | --- | --- |
| Source published in `fru180/Panestra-cli` | Public repository exists but reports size `0` | Pending push |
| `v0.1.0` GitHub Release | No releases returned | Pending tag/release |
| Homebrew `fru180/tap/panestra-cli` | `fru180/homebrew-tap` does not exist | Pending repository and formula publication |
| GitHub Topics | Repository topic list is empty | Pending metadata update |
| Panestra Web reciprocal link | Web README has no `Panestra-cli` reference | Pending cross-repository change |
| Terminal/agent hardware E2E matrix | Apple Silicon macOS、実Codex、tmux内外をローカル確認済み | Terminal.app/iTerm2の明示記録、Ghostty、Intel Mac、実Claude Codeの証跡が未完了 |

These rows require external state changes or hardware/software not present in the current workspace. Publication steps are in `docs/release.md`.
