package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/fru180/Panestra-cli/internal/config"
)

func TestToggleDoesNotOverwriteInvalidConfig(t *testing.T) {
	for _, command := range []string{"enable", "disable"} {
		t.Run(command, func(t *testing.T) {
			t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
			if err := os.MkdirAll(filepath.Dir(config.Path()), 0700); err != nil {
				t.Fatal(err)
			}
			invalid := []byte("enabled = false\nauto_tmux = nope\n")
			if err := os.WriteFile(config.Path(), invalid, 0600); err != nil {
				t.Fatal(err)
			}
			originalArgs := os.Args
			t.Cleanup(func() { os.Args = originalArgs })
			os.Args = []string{"panestra", command}

			if code := run(); code != 1 {
				t.Fatalf("run() = %d, want 1", code)
			}
			got, err := os.ReadFile(config.Path())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, invalid) {
				t.Fatalf("invalid config was overwritten:\n%s", got)
			}
		})
	}
}
