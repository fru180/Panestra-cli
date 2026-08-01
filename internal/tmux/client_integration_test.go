package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testClient(t *testing.T) (Client, string) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	tmuxTemp, err := os.MkdirTemp("/tmp", "pt-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmuxTemp) })
	t.Setenv("TMUX_TMPDIR", tmuxTemp)
	name := fmt.Sprintf("pc-%x", time.Now().UnixNano())
	bin := filepath.Join(t.TempDir(), "tmux-test")
	script := "#!/bin/sh\nexec tmux -L '" + name + "' \"$@\"\n"
	if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	c := Client{Binary: bin}
	if _, err := c.run("-f", "/dev/null", "new-session", "-d", "-s", "test"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = c.run("kill-server") })
	pane, err := c.run("display-message", "-p", "-t", "test", "#{pane_id}")
	if err != nil {
		t.Fatal(err)
	}
	return c, pane
}

func TestSetPromptSpecialCharacters(t *testing.T) {
	c, pane := testClient(t)
	want := "日本語 \" '$`;() #{pane_id}"
	if err := c.SetPrompt(pane, want); err != nil {
		t.Fatal(err)
	}
	got, err := c.run("show-option", "-pqv", "-t", pane, "@panestra_cli_prompt")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAcquireReleaseRestoresWindow(t *testing.T) {
	c, pane := testClient(t)
	beforeStatus, _ := c.run("show-options", "-wAv", "-t", pane, "pane-border-status")
	beforeFormat, _ := c.run("show-options", "-wAv", "-t", pane, "pane-border-format")
	if err := c.Acquire(pane, "Task: ", true); err != nil {
		t.Fatal(err)
	}
	status, _ := c.run("show-options", "-wAv", "-t", pane, "pane-border-status")
	if status != "top" {
		t.Fatalf("status = %q", status)
	}
	if err := c.Release(pane); err != nil {
		t.Fatal(err)
	}
	afterStatus, _ := c.run("show-options", "-wAv", "-t", pane, "pane-border-status")
	afterFormat, _ := c.run("show-options", "-wAv", "-t", pane, "pane-border-format")
	if afterStatus != beforeStatus || afterFormat != beforeFormat {
		t.Fatalf("not restored: %q/%q -> %q/%q", beforeStatus, beforeFormat, afterStatus, afterFormat)
	}
	active, _ := c.run("show-option", "-pqv", "-t", pane, "@panestra_cli_active")
	if strings.TrimSpace(active) != "" {
		t.Fatalf("active option remains: %q", active)
	}
}

func TestCaptureHistoryIncludesScrollbackAndExcludesBorder(t *testing.T) {
	c, pane := testClient(t)
	if err := c.Acquire(pane, "Task: ", true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Release(pane) })
	if err := c.SetPrompt(pane, "not-pane-output"); err != nil {
		t.Fatal(err)
	}
	command := "i=1; while [ $i -le 40 ]; do echo history-line-$i; i=$((i+1)); done"
	if _, err := c.run("send-keys", "-t", pane, command, "C-m"); err != nil {
		t.Fatal(err)
	}

	var history []byte
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var err error
		history, err = c.CaptureHistory(pane)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(history), "history-line-40") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	got := string(history)
	if !strings.Contains(got, "history-line-1") || !strings.Contains(got, "history-line-40") {
		t.Fatalf("history is incomplete: %q", got)
	}
	if strings.Contains(got, "Task: not-pane-output") {
		t.Fatalf("pane border leaked into history: %q", got)
	}
}
