package launcher

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
	if code := ExecStatus(status, encoded); code != 9 {
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
