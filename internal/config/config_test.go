package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndEnvironmentOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PANESTRA_CLI_HOME", home)
	if err := os.MkdirAll(filepath.Dir(Path()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(), []byte("enabled = true\nauto_tmux = false\nprefix = \"Work: #1 \"\nmax_width = 80\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PANESTRA_CLI_MAX_WIDTH", "40")
	t.Setenv("PANESTRA_CLI_PREFIX", "Now: ")
	t.Setenv("PANESTRA_CLI_DISABLE", "1")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Enabled || c.AutoTmux || c.Prefix != "Now: " || c.MaxWidth != 40 {
		t.Fatalf("unexpected config: %+v", c)
	}
}

func TestInvalidConfigReturnsSafeDisabledConfig(t *testing.T) {
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Dir(Path()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(), []byte("enabled = false\nauto_tmux = nope\nmax_width = 20\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil")
	}
	if got.Enabled {
		t.Fatalf("invalid config enabled Panestra CLI: %+v", got)
	}
	if !strings.Contains(err.Error(), Path()) {
		t.Fatalf("error %q does not contain config path %q", err, Path())
	}
}

func TestMissingConfigReturnsSafeDisabledConfig(t *testing.T) {
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	got, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil")
	}
	if got.Enabled {
		t.Fatalf("missing config enabled Panestra CLI: %+v", got)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	want := Config{Enabled: false, AutoTmux: true, Prefix: "Task: 日本語 ", MaxWidth: 42, ShowWaitingMessage: false}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
