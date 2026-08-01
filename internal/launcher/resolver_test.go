package launcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSkipsShim(t *testing.T) {
	home, realDir := t.TempDir(), t.TempDir()
	t.Setenv("PANESTRA_CLI_HOME", home)
	shimDir := filepath.Join(home, ".local", "share", "panestra-cli", "shims")
	if err := os.MkdirAll(shimDir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(shimDir, "codex"), filepath.Join(realDir, "codex")} {
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", shimDir+string(os.PathListSeparator)+realDir)
	got, err := Resolve("codex")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(realDir, "codex") {
		t.Fatalf("got %q", got)
	}
}
