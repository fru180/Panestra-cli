package launcher

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/fru180/Panestra-cli/internal/config"
)

func TestRunForwardsArgumentsAndExitCode(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "args")
	script := filepath.Join(dir, "agent")
	content := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$PANESTRA_TEST_CAPTURE\"\nexit 23\n"
	if err := os.WriteFile(script, []byte(content), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PANESTRA_TEST_CAPTURE", capture)
	if code := run(script, []string{"--model", "test model", "special '$`;()"}, 1); code != 23 {
		t.Fatalf("exit = %d", code)
	}
	b, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), "--model\ntest model\nspecial '$`;()\n"; got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
}

func TestExecStatusWritesAgentExitCode(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "agent")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 9\n"), 0700); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal([]string{script, "argument"})
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	status := filepath.Join(dir, "status")
	if code := ExecStatus(status, encoded, ""); code != 9 {
		t.Fatalf("exit = %d", code)
	}
	b, err := os.ReadFile(status)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "9" {
		t.Fatalf("status = %q", b)
	}
}

func TestExecStatusOwnsPaneOnlyWhileAgentRuns(t *testing.T) {
	tmux, pane := testLauncherTMUX(t)
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	if err := config.Save(config.Default()); err != nil {
		t.Fatal(err)
	}

	activeFile := filepath.Join(t.TempDir(), "active")
	t.Setenv("PANESTRA_TEST_ACTIVE", activeFile)
	agent := filepath.Join(t.TempDir(), "agent")
	script := `#!/bin/sh
tmux show-option -pqv -t "$TMUX_PANE" @panestra_cli_active > "$PANESTRA_TEST_ACTIVE"
tmux set-option -p -t "$TMUX_PANE" @panestra_cli_prompt "agent prompt"
exit 9
`
	if err := os.WriteFile(agent, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal([]string{agent})
	status := filepath.Join(t.TempDir(), "status")
	if code := ExecStatus(status, base64.RawURLEncoding.EncodeToString(payload), ""); code != 9 {
		t.Fatalf("exit = %d", code)
	}
	active, err := os.ReadFile(activeFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(active)) != "1" {
		t.Fatalf("agent observed active option %q", active)
	}
	for _, option := range []string{"@panestra_cli_active", "@panestra_cli_prompt"} {
		if value, _ := tmux("show-option", "-pqv", "-t", pane, option); value != "" {
			t.Fatalf("%s remains after normal exit: %q", option, value)
		}
	}
}

func TestExecStatusReleasesPaneAfterSignal(t *testing.T) {
	tmux, pane := testLauncherTMUX(t)
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	if err := config.Save(config.Default()); err != nil {
		t.Fatal(err)
	}

	ready := filepath.Join(t.TempDir(), "ready")
	t.Setenv("PANESTRA_TEST_READY", ready)
	agent := filepath.Join(t.TempDir(), "agent")
	script := `#!/bin/sh
trap 'exit 143' TERM
tmux set-option -p -t "$TMUX_PANE" @panestra_cli_prompt "agent prompt"
: > "$PANESTRA_TEST_READY"
while :; do sleep 1; done
`
	if err := os.WriteFile(agent, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal([]string{agent})
	status := filepath.Join(t.TempDir(), "status")
	done := make(chan int, 1)
	go func() {
		done <- ExecStatus(status, base64.RawURLEncoding.EncodeToString(payload), "")
	}()

	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("agent did not become ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-done:
		if code != 143 {
			t.Fatalf("exit = %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("agent did not exit after signal")
	}
	for _, option := range []string{"@panestra_cli_active", "@panestra_cli_prompt"} {
		if value, _ := tmux("show-option", "-pqv", "-t", pane, option); value != "" {
			t.Fatalf("%s remains after signal exit: %q", option, value)
		}
	}
}

func testLauncherTMUX(t *testing.T) (func(...string) (string, error), string) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	tmuxTemp, err := os.MkdirTemp("/tmp", "pl-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmuxTemp) })
	t.Setenv("TMUX_TMPDIR", tmuxTemp)
	name := fmt.Sprintf("pl-%x", time.Now().UnixNano())
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
	pane, err := tmux("display-message", "-p", "-t", "test", "#{pane_id}")
	if err != nil {
		t.Fatal(err)
	}
	socket, err := tmux("display-message", "-p", "-t", pane, "#{socket_path}")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMUX", socket+",0,0")
	t.Setenv("TMUX_PANE", pane)
	return tmux, pane
}

func TestDedicatedConfigEnablesScrollback(t *testing.T) {
	for _, setting := range []string{
		"set -g mouse on",
		"bind-key -n WheelUpPane copy-mode -e \\; send-keys -X -N 5 scroll-up",
	} {
		if !strings.Contains(dedicatedConfig, setting+"\n") {
			t.Fatalf("dedicated config is missing %q", setting)
		}
	}
}

func TestTranscriptSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.sock")
	listener, result, err := listenTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	want := []byte("first line\n日本語の最終出力\n")
	if err := sendTranscript(path, want); err != nil {
		t.Fatal(err)
	}
	got := <-result
	if got.err != nil {
		t.Fatal(got.err)
	}
	if !bytes.Equal(got.data, want) {
		t.Fatalf("transcript = %q, want %q", got.data, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("socket mode = %o", info.Mode().Perm())
	}
}

func TestTranscriptForTerminalRemovesBlankPaneRows(t *testing.T) {
	got := transcriptForTerminal([]byte("short output\n\n\n"))
	want := []byte("\x1b[0mshort output\n\x1b[0m")
	if !bytes.Equal(got, want) {
		t.Fatalf("terminal transcript = %q, want %q", got, want)
	}
	if got := transcriptForTerminal([]byte("\n\n")); got != nil {
		t.Fatalf("blank transcript = %q", got)
	}
}

func TestLaunchNoninteractivePreservesExitCode(t *testing.T) {
	home, bin := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte("#!/bin/sh\nexit 31\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PANESTRA_CLI_HOME", home)
	t.Setenv("PATH", bin)
	t.Setenv("PANESTRA_CLI_LAUNCH_DEPTH", "")
	if code := Launch("codex", []string{"--version"}); code != 31 {
		t.Fatalf("exit = %d", code)
	}
}

func TestLaunchRejectsRecursion(t *testing.T) {
	t.Setenv("PANESTRA_CLI_LAUNCH_DEPTH", "2")
	if code := Launch("codex", nil); code != 126 {
		t.Fatalf("exit = %d", code)
	}
}

func TestEnvironmentReplacement(t *testing.T) {
	got := withEnv([]string{"A=1", "DEPTH=old", "B=2"}, "DEPTH", "new")
	want := []string{"A=1", "B=2", "DEPTH=new"}
	if len(got) != len(want) {
		t.Fatalf("env = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("env = %v", got)
		}
	}
}

func TestShellQuote(t *testing.T) {
	if got, want := shellQuote("a'b c"), `'a'\''b c'`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
