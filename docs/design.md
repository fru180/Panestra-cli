# Panestra CLI v0.1.0 実装計画書

## 1. 概要

### 1.1 プロダクト名

**Panestra CLI**

### 1.2 リポジトリ

```text
https://github.com/fru180/Panestra-cli
```

Panestraプロジェクトは、次の2リポジトリで構成する。

```text
Panestraプロジェクト
├── fru180/Panestra
│   └── Webダッシュボード版
│
└── fru180/Panestra-cli
    └── 軽量なターミナル版
```

Panestra CLIは、Panestra Webを起動せず、Codex CLIまたはClaude Codeをターミナル上で軽量に利用したいユーザー向けのサブプロジェクトとする。

---

## 2. 目的

Codex CLIまたはClaude Codeへ送信した最新のプロンプトを、ターミナル画面内の上部へ固定表示する。

ユーザーはtmuxを意識せず、従来どおり次の操作だけを行う。

```text
ターミナルを起動
↓
codex または claude を実行
↓
プロンプトを入力して送信
↓
送信したプロンプトが画面上部へ固定表示される
```

表示イメージ：

```text
┌ Task: 認証処理をリファクタリングし、既存テストも修正してください ─────┐
│                                                                      │
│  Codex / Claude Code                                                 │
│                                                                      │
│  エージェントの応答や実行ログ                                       │
│                                                                      │
```

---

## 3. 解決する課題

AI駆動開発中に発生する、次の問題を軽減する。

* 長時間実行後に、何を指示していたか分からなくなる
* 元のプロンプトが出力に押し流されて画面外へ消える
* 複数のターミナルのうち、どれが何の作業か分からなくなる
* 追加指示を送った後、現在の作業内容を確認しづらい
* 完了した処理が、どの指示に基づくものか判断しにくい
* Webダッシュボードを起動するほどではない用途でも、現在の指示を常時確認したい

---

## 4. v0.1.0の対応範囲

### 4.1 正式対応環境

* macOS
* Apple Silicon Mac
* Intel Mac
* zsh
* Terminal.app
* iTerm2
* Ghostty
* tmux
* Codex CLI
* Claude Code

### 4.2 実装する機能

* 通常の`codex`コマンドを維持する
* 通常の`claude`コマンドを維持する
* tmux外では、内部的に1ペインのtmuxを自動起動する
* tmux内では、現在のtmuxペインを利用する
* 最新のユーザープロンプトを上部へ固定表示する
* 追加プロンプト送信時に表示を更新する
* 日本語を含む長文を表示幅に合わせて省略する
* エージェント終了後に元のシェルへ戻る
* 非対話コマンドは通常どおり実行する
* セットアップ状態を診断できる
* 一時的に無効化できる
* 完全にアンインストールできる
* プロンプトを永続保存しない
* 外部通信を行わない

### 4.3 v0.1.0では実装しない機能

* プロンプト履歴
* AIによるプロンプト要約
* 完了通知
* 入力待ち通知
* Gitブランチ表示
* 作業ディレクトリ表示
* 実行時間表示
* トークン使用量表示
* Panestra Webとの通信
* 複数エージェントの一覧管理
* Windows対応
* Linux正式対応
* bash正式対応
* Zellij対応

---

## 5. ユーザー体験

### 5.1 インストール

Homebrewを標準のインストール方法とする。

```bash
brew install fru180/tap/panestra-cli
panestra setup
```

名称は次のように統一する。

```text
製品名:          Panestra CLI
GitHubリポジトリ: Panestra-cli
Homebrew Formula: panestra-cli
管理コマンド:     panestra
通常利用コマンド: codex / claude
```

### 5.2 初期設定

```bash
panestra setup
```

`setup`は次の処理を行う。

1. macOSであることを確認する
2. 使用中のシェルを確認する
3. tmuxの存在を確認する
4. Codex CLIの存在を確認する
5. Claude Codeの存在を確認する
6. Panestra CLI用shimを作成する
7. shimディレクトリをPATHの先頭へ追加する
8. Codex用フックを導入する
9. Claude Code用フックを導入する
10. 設定ファイルを作成する
11. `panestra doctor`を実行する

CodexまたはClaude Codeの一方しかインストールされていない場合は、存在するエージェントだけを設定する。

`setup`を複数回実行しても、設定やPATHが重複しないようにする。

### 5.3 通常利用

セットアップ後も従来と同じコマンドを利用する。

```bash
codex
```

または、

```bash
claude
```

起動直後：

```text
┌ Panestra CLI — Waiting for prompt ────────────────────────────────────┐
│                                                                      │
│  Codex                                                               │
```

プロンプト送信後：

```text
┌ Task: ログイン処理をJWT認証へ変更してください ───────────────────────┐
│                                                                      │
│  Codex                                                               │
```

追加指示送信後：

```text
┌ Task: 既存テストも新しい認証方式へ合わせて修正してください ─────────┐
│                                                                      │
│  Codex                                                               │
```

表示内容は常に最新のユーザープロンプトへ置き換える。

### 5.4 一時的な無効化

1回だけ無効化する場合：

```bash
PANESTRA_CLI_DISABLE=1 codex
```

継続的に無効化する場合：

```bash
panestra disable
```

再度有効化する場合：

```bash
panestra enable
```

### 5.5 表示内容の消去

```bash
panestra clear
```

現在のtmuxペインに設定されているプロンプトを消去する。

---

## 6. システム構成

Panestra CLIは、次の2要素で構成する。

```text
Panestra CLI
├── ネイティブCLI
│   ├── codex / claude起動前の処理
│   ├── tmuxの自動起動
│   ├── shim管理
│   ├── セットアップ
│   ├── 診断
│   ├── アンインストール
│   └── プロンプト表示処理
│
└── エージェントアダプター
    ├── Codex用フック
    └── Claude Code用フック
```

処理フロー：

```text
ユーザー
  │
  │ codex
  ▼
Panestra CLI shim
  │
  ├── tmux内
  │     └── 現在のペインを利用
  │
  └── tmux外
        └── 透明な1ペインtmuxを起動
                │
                ▼
          Codex CLI起動
                │
                │ プロンプト送信
                ▼
        UserPromptSubmitフック
                │
                ▼
          panestra hook
                │
                ▼
        tmuxペインオプション更新
                │
                ▼
        ペイン上部の表示を更新
```

エージェントのフックは起動後に実行されるため、tmuxの自動起動はフックではなく、起動前に動作するshimが担当する。

---

## 7. 実装言語

### 7.1 コアCLI

**Go**で実装する。

採用理由：

* 単一バイナリとして配布できる
* Node.jsやPythonを要求しない
* Homebrewで配布しやすい
* Apple SiliconとIntel Macの両方へビルドできる
* JSON解析を標準ライブラリで処理できる
* プロセス起動と引数転送を安全に実装できる
* シェルスクリプトよりPATH探索や再帰起動を安全に扱える
* フックからの起動が高速
* 外部依存を最小限にできる

### 7.2 エージェントアダプター

Codex用とClaude Code用のアダプターには、複雑な処理を持たせない。

アダプターは、`UserPromptSubmit`イベント発生時に次のコマンドを呼び出す。

```text
panestra hook
```

プロンプトの解析、整形、tmux操作はすべてGo製CLI側へ集約する。

---

## 8. リポジトリ構成

```text
Panestra-cli/
├── cmd/
│   └── panestra/
│       └── main.go
│
├── internal/
│   ├── agent/
│   │   ├── agent.go
│   │   ├── codex.go
│   │   └── claude.go
│   │
│   ├── config/
│   │   ├── config.go
│   │   └── default.go
│   │
│   ├── hook/
│   │   ├── event.go
│   │   ├── parser.go
│   │   └── handler.go
│   │
│   ├── installer/
│   │   ├── setup.go
│   │   ├── uninstall.go
│   │   ├── shell.go
│   │   └── adapter.go
│   │
│   ├── launcher/
│   │   ├── launcher.go
│   │   ├── resolver.go
│   │   ├── interactive.go
│   │   └── environment.go
│   │
│   ├── prompt/
│   │   ├── normalize.go
│   │   ├── sanitize.go
│   │   └── truncate.go
│   │
│   └── tmux/
│       ├── client.go
│       ├── session.go
│       ├── renderer.go
│       └── restore.go
│
├── adapters/
│   ├── codex/
│   │   ├── manifest/
│   │   └── hooks/
│   │       └── hooks.json
│   │
│   └── claude/
│       ├── manifest/
│       └── hooks/
│           └── hooks.json
│
├── assets/
│   └── panestra-cli.tmux.conf
│
├── scripts/
│   ├── install.sh
│   └── uninstall.sh
│
├── docs/
│   ├── architecture.md
│   ├── installation.md
│   └── troubleshooting.md
│
├── .github/
│   └── workflows/
│       ├── test.yml
│       ├── integration.yml
│       └── release.yml
│
├── go.mod
├── go.sum
├── README.md
├── LICENSE
├── SECURITY.md
├── CONTRIBUTING.md
└── CHANGELOG.md
```

CodexとClaude Codeでフック仕様やインストール方法が異なる場合に備え、アダプターを分離する。

---

## 9. CLIコマンド設計

```text
panestra setup
panestra doctor
panestra uninstall
panestra enable
panestra disable
panestra clear
panestra hook
panestra launch <agent> [arguments...]
panestra version
```

### 9.1 `panestra setup`

初期設定を行う。

主な処理：

* 対応OS確認
* シェル確認
* tmux確認
* Codex確認
* Claude Code確認
* shim作成
* PATH設定
* アダプター導入
* 設定ファイル作成
* doctor実行

### 9.2 `panestra doctor`

導入状態を診断する。

表示例：

```text
Panestra CLI Doctor

✓ Panestra CLI 0.1.0
✓ macOS detected
✓ zsh detected
✓ tmux detected
✓ Codex CLI detected
✓ Claude Code detected
✓ shim directory is in PATH
✓ codex shim is active
✓ claude shim is active
✓ Codex adapter is installed
✓ Claude Code adapter is installed
✓ Panestra CLI is enabled
```

問題がある場合：

```text
✗ tmux was not found
  Run: brew install tmux

✗ Shell configuration has not been loaded
  Run: source ~/.zshrc
```

### 9.3 `panestra hook`

エージェントのフックから呼ばれる内部コマンド。

標準入力例：

```json
{
  "hook_event_name": "UserPromptSubmit",
  "prompt": "認証処理を修正してください"
}
```

処理内容：

1. 標準入力からJSONを読み込む
2. `prompt`を取得する
3. プロンプトを正規化する
4. 表示幅に合わせて省略する
5. 対象tmuxペインを特定する
6. tmuxペインオプションを更新する
7. 終了コード0で終了する

フック処理では標準出力へ何も書き込まない。

### 9.4 `panestra launch`

shimから呼び出される内部コマンド。

```bash
panestra launch codex
panestra launch claude
```

処理内容：

* 本物のエージェント実行ファイルを探索する
* 対話モードか判定する
* tmux内外を判定する
* 必要に応じて専用tmuxを起動する
* すべての引数をエージェントへ転送する
* エージェントの終了コードをそのまま返す

### 9.5 `panestra uninstall`

次の設定を削除する。

* Codex用アダプター
* Claude Code用アダプター
* `codex`用shim
* `claude`用shim
* `.zshrc`内のPanestra CLI管理ブロック
* Panestra CLI設定ファイル
* Panestra CLI用データディレクトリ

Codex CLI、Claude Code、tmux本体は削除しない。

---

## 10. shim設計

セットアップ時に次のディレクトリを作成する。

```text
~/.local/share/panestra-cli/shims/
├── codex
└── claude
```

`.zshrc`へ次のブロックを追加する。

```bash
# >>> panestra-cli >>>
export PATH="$HOME/.local/share/panestra-cli/shims:$PATH"
# <<< panestra-cli <<<
```

ユーザーが次を実行する。

```bash
codex
```

内部では次の処理になる。

```text
codex shim
↓
panestra launch codex
↓
本物のcodex
```

本物の実行ファイルを探索するときは、Panestra CLIのshimディレクトリをPATH検索対象から除外する。

再帰起動を防ぐため、次も確認する。

* shimと候補実行ファイルの実体パスが同じでないこと
* `PANESTRA_CLI_LAUNCH_DEPTH`が規定値を超えていないこと
* 同じPanestra CLIバイナリをエージェント本体として選択していないこと

---

## 11. 対話モード判定

通常の対話TUIを起動する場合だけ、tmuxを自動起動する。

### 11.1 tmuxを使用する例

```bash
codex
claude
codex --model MODEL
claude --model MODEL
```

### 11.2 Panestra CLIをバイパスする例

```bash
codex --help
codex --version
codex exec "テストを実行して"
codex review
claude --help
claude --version
claude -p "要約してください"
echo "質問" | claude
```

判定材料：

* stdinがTTYか
* stdoutがTTYか
* 非対話用サブコマンドが指定されているか
* 非対話用オプションが指定されているか
* `PANESTRA_CLI_DISABLE`が設定されているか

判定できない場合は、tmuxを挟まずエージェント本体を通常起動する。

---

## 12. tmux起動設計

### 12.1 tmux外から起動した場合

Panestra CLI専用のtmuxサーバーを起動する。

概念例：

```bash
tmux \
  -L panestra-cli \
  -f /path/to/panestra-cli.tmux.conf \
  new-session \
  <agent-command>
```

専用tmuxサーバーを使うことで、ユーザーが普段使用しているtmuxセッションや設定との干渉を避ける。

セッション名には衝突しない値を使用する。

```text
panestra-cli-<PID>-<random>
```

エージェント終了時にセッションも終了し、元のシェルへ戻る。

### 12.2 専用tmux設定

```tmux
set -g status off
set -g pane-border-status top
set -g pane-border-format '#{?#{@panestra_cli_prompt}, Task: #{@panestra_cli_prompt}, Panestra CLI — Waiting for prompt }'
set -g allow-rename off
set -g set-titles off
```

方針：

* 下部ステータスバーを表示しない
* 上部の1行だけを使用する
* 1ペインで起動する
* tmux固有の表示を最小限にする
* エージェント終了時にtmuxも終了する

### 12.3 既存tmux内から起動した場合

`TMUX`環境変数が存在する場合は、tmuxを二重起動しない。

現在のペインを利用し、必要な表示設定だけを一時適用する。

適用前の値を保存する。

```text
@panestra_cli_previous_border_status
@panestra_cli_previous_border_format
@panestra_cli_refcount
```

最後のPanestra CLI利用プロセスが終了した時点で、元の設定へ戻す。

既存tmux環境で設定の安全な復元が保証できない場合は、設定を永続変更せず、警告を表示して通常起動する。

---

## 13. ペイン表示設計

通常のtmuxペインタイトルではなく、tmuxのユーザーオプションを利用する。

```text
@panestra_cli_prompt
```

Goからは、シェルを経由せず引数配列としてtmuxを実行する。

```go
exec.Command(
    "tmux",
    "set-option",
    "-p",
    "-t",
    paneID,
    "@panestra_cli_prompt",
    title,
)
```

これにより、プロンプトに次の文字が含まれていてもシェルコマンドとして評価されない。

```text
"
'
;
$
`
改行
括弧
日本語
絵文字
```

対象ペインは、可能な限り`TMUX_PANE`環境変数を使って特定する。

現在選択中のペインを対象にすると、フック実行中にユーザーが別ペインへ移動した場合に誤更新する可能性があるため使用しない。

---

## 14. プロンプト整形仕様

入力例：

```text
ユーザー登録APIを実装してください。

条件:
- メールアドレス認証
- 重複チェック
- テスト追加
```

表示例：

```text
Task: ユーザー登録APIを実装してください。 条件: - メールアドレス認証 - 重複チェック - テスト追加
```

### 14.1 整形規則

* 前後の空白を削除する
* 改行を半角スペースへ変換する
* タブを半角スペースへ変換する
* 連続する空白を1つへ統合する
* ANSIエスケープシーケンスを削除する
* 制御文字を削除する
* NUL文字を削除する
* 空文字の場合は表示を変更しない
* 長文は末尾を`…`にして省略する
* 日本語の全角文字幅を考慮する
* 絵文字や結合文字の途中で切断しない

### 14.2 最大表示幅

設定値の初期値は120カラムとする。

実際の表示幅は、次の小さい方を使用する。

```text
設定された最大幅
現在のtmuxペイン幅から装飾部分を引いた幅
```

例：

```text
ペイン幅: 100
装飾部分: 8
表示可能幅: 92
設定値: 120

実際の最大表示幅: 92
```

---

## 15. 設定ファイル

設定ファイル：

```text
~/.config/panestra-cli/config.toml
```

初期設定：

```toml
enabled = true
auto_tmux = true
prefix = "Task: "
max_width = 120
show_waiting_message = true
```

環境変数：

```text
PANESTRA_CLI_DISABLE
PANESTRA_CLI_MAX_WIDTH
PANESTRA_CLI_PREFIX
PANESTRA_CLI_DEBUG
```

優先順位：

```text
環境変数
↓
設定ファイル
↓
デフォルト値
```

---

## 16. エラー処理

Panestra CLIは補助機能であり、障害によってCodexやClaude Codeの利用を妨げてはならない。

### 16.1 フェイルオープン方針

| 状況              | 挙動                         |
| --------------- | -------------------------- |
| tmuxが見つからない     | 警告後、エージェントを通常起動            |
| 設定ファイルが不正       | デフォルト設定を使用                 |
| フックJSONが不正      | 何もせず終了コード0                 |
| `prompt`がない     | 何もせず終了コード0                 |
| `TMUX_PANE`がない  | 何もせず終了コード0                 |
| tmux更新に失敗       | エージェント処理を継続                |
| shimから本体を発見できない | 明確なエラーを表示して終了              |
| 対話モード判定に失敗      | tmuxを挟まず通常起動               |
| 設定復元に失敗         | stderrへ警告し、エージェントの終了コードは保持 |

### 16.2 フック処理

* 原則として終了コード0を返す
* 標準出力には何も書かない
* エラーやデバッグ情報はstderrへ出す
* 通常時はstderrにも出さない
* 1秒以内に完了する
* tmux操作が失敗しても再試行を繰り返さない

---

## 17. プライバシーとセキュリティ

### 17.1 基本方針

* プロンプトをファイルへ保存しない
* プロンプト履歴を作成しない
* 外部APIへ送信しない
* AIによる要約を行わない
* テレメトリーを送信しない
* ネットワーク接続を必要としない
* tmuxのペインオプションにだけ一時保持する

### 17.2 画面共有時の注意

最新プロンプトがターミナル上部へ表示されるため、画面共有や録画に含まれる。

READMEと初回セットアップ時に次の注意を表示する。

```text
Panestra CLI displays your latest prompt on screen.
Prompts may be visible during screen sharing or recording.
```

### 17.3 コマンドインジェクション対策

プロンプトをシェルコマンドとして評価しない。

禁止例：

```go
exec.Command("sh", "-c", "tmux set-option ... '"+prompt+"'")
```

採用例：

```go
exec.Command(
    "tmux",
    "set-option",
    "-p",
    "-t",
    paneID,
    "@panestra_cli_prompt",
    prompt,
)
```

### 17.4 ファイル変更

`panestra setup`が変更するファイルは次に限定する。

* `~/.zshrc`
* `~/.config/panestra-cli/config.toml`
* `~/.local/share/panestra-cli/`
* Codexのフック／プラグイン設定
* Claude Codeのフック／プラグイン設定

既存ファイルは変更前に内容を読み込み、Panestra CLIが管理する範囲だけを追加・削除する。

---

## 18. Panestra Webとの関係表示

### 18.1 Panestra CLI側README

冒頭に次を記載する。

```text
Panestra CLI is the lightweight, terminal-native edition of Panestra.

It keeps the latest prompt visible at the top of your terminal while Codex CLI
or Claude Code is running, without requiring the Panestra web dashboard.
```

日本語：

```text
Panestra CLIは、Panestraの軽量なターミナル版です。

Panestra Webを起動せず、Codex CLIやClaude Codeへ送信した最新の指示を
ターミナル上部へ固定表示します。
```

Web版へのリンクを掲載する。

```text
https://github.com/fru180/Panestra
```

### 18.2 Panestra Web側README

CLI版へのリンクを掲載する。

```text
https://github.com/fru180/Panestra-cli
```

説明：

```text
For a lightweight terminal-only experience without the web dashboard,
see Panestra CLI.
```

### 18.3 GitHub Topics

共通Topics：

```text
panestra
ai-coding-agent
codex-cli
claude-code
developer-tools
```

Panestra CLI固有Topics：

```text
tmux
terminal
cli
macos
```

---

## 19. テスト計画

### 19.1 単体テスト

対象：

* Codex形式JSONの解析
* Claude Code形式JSONの解析
* 不正JSON
* `prompt`欠落
* 空プロンプト
* 日本語の表示幅計算
* 絵文字の表示幅計算
* 結合文字の処理
* 改行の正規化
* タブの正規化
* 連続空白の統合
* ANSIシーケンス除去
* 制御文字除去
* 長文省略
* 設定ファイル解析
* 環境変数による設定上書き
* 実行ファイル探索
* shim自身の除外
* 再帰起動検出
* 引数転送
* 対話モード判定

### 19.2 tmux統合テスト

テスト専用のtmuxサーバーを起動して検証する。

確認項目：

* 正しいペインにプロンプトが設定される
* 他のペインを変更しない
* 追加プロンプトで更新される
* `panestra clear`で表示が消える
* 特殊文字を安全に扱える
* 複数セッションが干渉しない
* エージェント終了後に専用セッションが残らない
* 既存tmux設定が復元される
* tmux更新失敗時もフックが成功扱いになる

### 19.3 E2Eテスト

対象環境：

* Terminal.app
* iTerm2
* Ghostty
* Apple Silicon Mac
* Intel Mac
* Codex CLI
* Claude Code
* tmux外
* tmux内

基本シナリオ：

1. ターミナルを起動する
2. `codex`を実行する
3. プロンプトを一度入力する
4. 上部へ同じ内容が表示される
5. 追加指示を送信する
6. 上部が最新指示へ更新される
7. Codexを終了する
8. 元のシェルへ戻る
9. 専用tmuxセッションが残っていないことを確認する
10. Codexの終了コードが保持されていることを確認する

Claude Codeでも同じシナリオを実施する。

### 19.4 非対話コマンドテスト

次のコマンドがtmuxを起動せず、従来どおり動作することを確認する。

```bash
codex --help
codex --version
codex exec "test"
codex review
claude --help
claude --version
claude -p "test"
```

### 19.5 セットアップテスト

確認項目：

* 初回セットアップが成功する
* `setup`を2回実行しても設定が重複しない
* Codexだけ存在する環境で成功する
* Claude Codeだけ存在する環境で成功する
* 両方存在する環境で成功する
* tmuxがない場合に案内を表示する
* `.zshrc`の既存内容を壊さない
* PATHの反映前後を正しく診断する

### 19.6 アンインストールテスト

確認項目：

* shimが削除される
* `.zshrc`の管理ブロックが削除される
* Codexアダプターが削除される
* Claude Codeアダプターが削除される
* 設定ファイルが削除される
* 通常の`codex`と`claude`へ戻る
* 既存のユーザー設定を壊さない
* 複数回アンインストールしても問題が起きない

---

## 20. 実装フェーズ

### Phase 0：技術検証

実施内容：

* Codexのプロンプト送信イベントを取得する
* Claude Codeのプロンプト送信イベントを取得する
* フックからtmuxペインオプションを更新する
* `pane-border-status top`で上部表示する
* Terminal.app上で動作確認する
* CodexとClaude Codeの入力JSON形式を保存する
* フック導入手順を確認する

完了条件：

```text
既存のtmux内でCodexまたはClaude Codeへプロンプトを一度入力すると、
同じ内容がペイン上部へ表示される
```

### Phase 1：Go製コアCLI

実装内容：

* Goプロジェクト作成
* CLI基盤実装
* `panestra hook`
* フックJSON解析
* プロンプト整形
* 表示幅計算
* tmux操作
* 設定ファイル読み込み
* 単体テスト
* tmux統合テスト

完了条件：

```bash
printf '%s' '{"prompt":"認証処理を修正"}' | panestra hook
```

によって、指定されたtmuxペインの表示内容が更新される。

### Phase 2：透明tmuxランチャー

実装内容：

* `panestra launch`
* 本物のCodex／Claude Code探索
* shim作成
* `argv[0]`または引数によるエージェント判別
* TTY判定
* 非対話コマンドのバイパス
* 専用tmuxサーバー起動
* 既存tmux内での処理
* 引数転送
* 環境変数転送
* シグナル転送
* 終了コード転送
* 設定復元

完了条件：

```bash
codex
```

だけで内部的にtmuxが起動し、ユーザーがtmuxを直接操作せずCodexを利用できる。

### Phase 3：Codex／Claude Codeアダプター

実装内容：

* Codex用フック定義
* Claude Code用フック定義
* `panestra hook`の呼び出し
* アダプターのインストール処理
* アダプターの更新処理
* アダプターの削除処理
* 両エージェントでの動作確認

完了条件：

```text
プロンプト送信
↓
UserPromptSubmit
↓
panestra hook
↓
上部表示更新
```

がCodexとClaude Codeの両方で動作する。

### Phase 4：セットアップと管理コマンド

実装内容：

* `panestra setup`
* `panestra doctor`
* `panestra uninstall`
* `panestra enable`
* `panestra disable`
* `panestra clear`
* `.zshrc`管理
* shim管理
* 設定ファイル管理
* 冪等性
* エラー修復

完了条件：

* `setup`を複数回実行しても設定が重複しない
* `doctor`で導入上の問題を特定できる
* `uninstall`で導入前の状態へ戻せる

### Phase 5：公開準備

実装内容：

* README
* デモGIF
* MIT License
* SECURITY.md
* CONTRIBUTING.md
* CHANGELOG.md
* GitHub Actions
* GitHub Releases
* Homebrew Tap
* インストール手順
* アンインストール手順
* トラブルシューティング
* Panestra Webとの相互リンク

完了条件：

第三者が次の手順だけで導入できる。

```bash
brew install fru180/tap/panestra-cli
panestra setup
codex
```

---

## 21. GitHub Actions

### 21.1 Pull Request時

実行項目：

```text
go test ./...
go vet ./...
gofmt確認
静的解析
tmux統合テスト
アダプター定義の検証
インストールスクリプトの構文確認
```

### 21.2 リリース時

`v0.1.0`タグ作成時に次を実行する。

1. macOS Apple Silicon向けバイナリをビルドする
2. macOS Intel向けバイナリをビルドする
3. tar.gzを生成する
4. SHA-256チェックサムを生成する
5. GitHub Releaseを作成する
6. Homebrew Formulaを更新する

生成物：

```text
panestra-cli_Darwin_arm64.tar.gz
panestra-cli_Darwin_x86_64.tar.gz
checksums.txt
```

---

## 22. リリース構成

### 22.1 GitHub

```text
https://github.com/fru180/Panestra-cli
```

### 22.2 Homebrew

```bash
brew install fru180/tap/panestra-cli
```

Homebrew Formulaではtmuxを依存関係として指定する。

```text
panestra-cli
└── tmux
```

### 22.3 インストール後のコマンド

管理用：

```bash
panestra setup
panestra doctor
panestra uninstall
```

通常利用：

```bash
codex
claude
```

---

## 23. v0.1.0の完了条件

以下をすべて満たした時点で、Panestra CLI v0.1.0を完成とする。

### 起動

* ユーザーがtmuxを直接起動する必要がない
* ユーザーがtmuxのキー操作を覚える必要がない
* `codex`を従来どおり実行できる
* `claude`を従来どおり実行できる
* tmux内では二重にtmuxを起動しない

### プロンプト表示

* プロンプトを一度だけ入力すればよい
* 送信直後に最新プロンプトが上部へ表示される
* 追加指示で表示内容が更新される
* 日本語を正しく表示できる
* 長文を表示幅に合わせて省略できる
* 特殊文字を含むプロンプトを安全に処理できる

### 互換性

* Codex CLIで動作する
* Claude Codeで動作する
* 非対話コマンドを妨げない
* エージェントの引数をそのまま転送できる
* エージェントの終了コードを保持できる
* エージェント終了後に元のシェルへ戻る
* 専用tmuxセッションが不要に残らない

### 安全性

* Panestra CLIの障害がエージェントのプロンプト送信を止めない
* プロンプトを永続保存しない
* 外部通信を行わない
* シェル経由でプロンプトを評価しない
* 既存のtmux設定を永続変更しない
* アンインストールで導入前の状態へ戻せる

### 配布

* GitHubでソースコードを公開する
* GitHub Releaseからバイナリを取得できる
* Homebrewからインストールできる
* READMEだけで第三者が導入できる
* Panestra Webとの関係がREADMEで明確になっている
