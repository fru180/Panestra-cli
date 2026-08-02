package hook

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/fru180/Panestra-cli/internal/config"
	"github.com/fru180/Panestra-cli/internal/prompt"
	ptmux "github.com/fru180/Panestra-cli/internal/tmux"
)

func TestHandleUpdatesOnlyTargetPane(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	tmuxTemp, err := os.MkdirTemp("/tmp", "pt-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmuxTemp) })
	t.Setenv("TMUX_TMPDIR", tmuxTemp)
	name := fmt.Sprintf("ph-%x", time.Now().UnixNano())
	tmux := func(args ...string) (string, error) {
		cmd := exec.Command("tmux", append([]string{"-L", name}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return strings.TrimSpace(string(out)), nil
	}
	if _, err := tmux("-f", "/dev/null", "new-session", "-d", "-s", "test", "sleep 60"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })
	if _, err := tmux("split-window", "-d", "-t", "test", "sleep 60"); err != nil {
		t.Fatal(err)
	}
	panes, err := tmux("list-panes", "-t", "test", "-F", "#{pane_id}")
	if err != nil {
		t.Fatal(err)
	}
	ids := strings.Fields(panes)
	if len(ids) != 2 {
		t.Fatalf("panes = %q", panes)
	}
	socket, err := tmux("display-message", "-p", "-t", ids[0], "#{socket_path}")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMUX", socket+",0,0")
	t.Setenv("TMUX_PANE", ids[0])
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	if err := config.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	if _, err := tmux("resize-window", "-t", "test", "-x", "40"); err != nil {
		t.Fatal(err)
	}

	Handle(strings.NewReader(`{"hook_event_name":"UserPromptSubmit","prompt":"所有されていないprompt"}`))
	for _, option := range []string{"@panestra_cli_prompt", "@panestra_cli_prefix"} {
		if value, _ := tmux("show-options", "-pqv", "-t", ids[0], option); value != "" {
			t.Fatalf("%s was saved without active ownership: %q", option, value)
		}
	}

	client := ptmux.New()
	if err := client.Acquire(ids[0], "Task: ", true); err != nil {
		t.Fatal(err)
	}
	Handle(strings.NewReader("{\"hook_event_name\":\"UserPromptSubmit\",\"prompt\":\"認証処理を修正してください。\\n条件: '$` ; () 日本語の長い追加条件\"}"))
	got, err := tmux("show-options", "-pqv", "-t", ids[0], "@panestra_cli_prompt")
	if err != nil {
		t.Fatal(err)
	}
	if got == "" || !strings.HasPrefix(got, "認証処理") {
		t.Fatalf("prompt = %q", got)
	}
	if prompt.Width(got) > 32 {
		t.Fatalf("prompt width = %d, value = %q", prompt.Width(got), got)
	}
	prefix, err := tmux("show-options", "-pqv", "-t", ids[0], "@panestra_cli_prefix")
	if err != nil {
		t.Fatal(err)
	}
	if prefix != "Task:" && prefix != "Task: " {
		t.Fatalf("prefix = %q", prefix)
	}
	other, _ := tmux("show-options", "-pqv", "-t", ids[1], "@panestra_cli_prompt")
	if other != "" {
		t.Fatalf("other pane was modified: %q", other)
	}

	Handle(strings.NewReader(`{"hook_event_name":"UserPromptSubmit","prompt":"追加指示"}`))
	updated, err := tmux("show-options", "-pqv", "-t", ids[0], "@panestra_cli_prompt")
	if err != nil {
		t.Fatal(err)
	}
	if updated != "追加指示" {
		t.Fatalf("prompt was not replaced: %q", updated)
	}

	for _, option := range []string{"@panestra_cli_prompt", "@panestra_cli_prefix"} {
		if _, err := tmux("set-option", "-pu", "-t", ids[0], option); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(config.Path(), []byte("enabled = false\nauto_tmux = nope\nmax_width = 20\n"), 0600); err != nil {
		t.Fatal(err)
	}
	Handle(strings.NewReader(`{"hook_event_name":"UserPromptSubmit","prompt":"保存してはいけない"}`))
	for _, option := range []string{"@panestra_cli_prompt", "@panestra_cli_prefix"} {
		if value, _ := tmux("show-options", "-pqv", "-t", ids[0], option); value != "" {
			t.Fatalf("%s was saved for invalid config: %q", option, value)
		}
	}

	if err := client.Release(ids[0]); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	Handle(strings.NewReader(`{"hook_event_name":"UserPromptSubmit","prompt":"解放後のprompt"}`))
	if value, _ := tmux("show-options", "-pqv", "-t", ids[0], "@panestra_cli_prompt"); value != "" {
		t.Fatalf("prompt was saved after release: %q", value)
	}
}

func TestHandleFailsOpen(t *testing.T) {
	t.Setenv("TMUX_PANE", "%99999")
	Handle(strings.NewReader("{invalid"))
	Handle(strings.NewReader(`{"hook_event_name":"UserPromptSubmit"}`))
	Handle(strings.NewReader(`{"hook_event_name":"UserPromptSubmit","prompt":""}`))
}
