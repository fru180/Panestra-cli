package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fru180/Panestra-cli/internal/config"
	"github.com/fru180/Panestra-cli/internal/launcher"
)

func Doctor() bool {
	fmt.Println("Panestra CLI Doctor")
	ok := true
	check := func(good bool, yes, no string) {
		if good {
			fmt.Println("✓", yes)
		} else {
			fmt.Println("✗", no)
			ok = false
		}
	}
	check(true, "Panestra CLI "+config.Version, "")
	check(runtime.GOOS == "darwin", "macOS detected", "unsupported OS: "+runtime.GOOS)
	check(filepath.Base(os.Getenv("SHELL")) == "zsh", "zsh detected", "zsh is not the current shell")
	_, err := exec.LookPath("tmux")
	check(err == nil, "tmux detected", "tmux was not found (run: brew install tmux)")
	found := 0
	for _, agent := range []string{"codex", "claude"} {
		_, err := launcher.Resolve(agent)
		exists := err == nil
		if exists {
			found++
			check(true, agent+" CLI detected", "")
			check(IsShimActive(agent), agent+" shim is active", agent+" shim is not first in PATH (run: source ~/.zshrc)")
			adapterPath := codexHooksPath()
			if agent == "claude" {
				adapterPath = claudeSettingsPath()
			}
			check(adapterInstalled(adapterPath), agent+" adapter is installed", agent+" adapter is missing (run: panestra setup)")
		} else {
			fmt.Println("-", agent+" CLI is not installed (optional)")
		}
	}
	check(found > 0, "at least one supported agent detected", "neither Codex CLI nor Claude Code was found")
	pathActive := false
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == config.ShimDir() {
			pathActive = true
		}
	}
	check(pathActive, "shim directory is in PATH", "shim directory is not in PATH (run: source ~/.zshrc)")
	check(config.Load().Enabled, "Panestra CLI is enabled", "Panestra CLI is disabled (run: panestra enable)")
	if b, err := os.ReadFile(zshrcPath()); err == nil {
		check(strings.Count(string(b), beginMarker) == 1, "shell configuration is installed once", "shell configuration is missing or duplicated")
	}
	return ok
}
