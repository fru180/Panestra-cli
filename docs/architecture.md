# Architecture

Panestra CLI は単一の Go バイナリ、`codex` / `claude` shim、両エージェントの `UserPromptSubmit` フックから構成されます。

```text
codex / claude shim
  -> panestra launch <agent>
     -> 対話TTY: 既存tmuxペイン、または専用tmuxサーバー
     -> 非対話: エージェントを直接実行

UserPromptSubmit (JSON on stdin)
  -> panestra hook
     -> 正規化・表示幅で切り詰め
     -> tmux @panestra_cli_prompt を更新
```

専用 tmux サーバーはユーザーの通常の tmux サーバーとソケットを分離します。既存 tmux 内ではウィンドウの border 設定を保存し、参照カウントがゼロになったときに復元します。対象ペインは選択状態ではなく `TMUX_PANE` で特定します。

プロンプトはシェルへ連結せず、`exec.Command` の引数として tmux へ渡します。ANSI シーケンスと制御文字を除去し、Unicode 書記素クラスタを分割せずに端末セル幅で省略します。

セットアップは既存 JSON 設定を構造として読み、Panestra CLI のハンドラーだけを追加・更新します。アンインストールも同じ識別子のハンドラーと `.zshrc` の管理ブロックだけを削除します。
