package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fru180/Panestra-cli/internal/config"
	"github.com/fru180/Panestra-cli/internal/launcher"
)

func Doctor() bool {
	return doctor(os.Stdout)
}

func doctor(out io.Writer) bool {
	fmt.Fprintln(out, "Panestra CLI Doctor")
	ok := true
	check := func(good bool, yes, no string) {
		if good {
			fmt.Fprintln(out, "✓", yes)
		} else {
			fmt.Fprintln(out, "✗", no)
			ok = false
		}
	}
	check(true, "Panestra CLI "+config.Version, "")
	check(runtime.GOOS == "darwin", "macOS detected", "unsupported OS: "+runtime.GOOS)
	check(filepath.Base(os.Getenv("SHELL")) == "zsh", "zsh detected", "zsh is not the current shell")
	_, err := exec.LookPath("tmux")
	check(err == nil, "tmux detected", "tmux was not found (run: brew install tmux)")
	writeTerminalHints(out)
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
			fmt.Fprintln(out, "-", agent+" CLI is not installed (optional)")
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
	cfg, configErr := config.Load()
	if configErr != nil {
		check(false, "", fmt.Sprintf("configuration could not be loaded: %v (fix %s, then run: panestra enable)", configErr, config.Path()))
	} else {
		check(true, "configuration loaded: "+config.Path(), "")
		check(cfg.Enabled, "Panestra CLI is enabled", "Panestra CLI is disabled (run: panestra enable)")
	}
	if b, err := os.ReadFile(zshrcPath()); err == nil {
		check(strings.Count(string(b), beginMarker) == 1, "shell configuration is installed once", "shell configuration is missing or duplicated")
	}
	return ok
}

func writeTerminalHints(out io.Writer) {
	if os.Getenv("TERM_PROGRAM") != "iTerm.app" && os.Getenv("LC_TERMINAL") != "iTerm2" {
		return
	}
	fmt.Fprintln(out, "- iTerm2 scrolling requires Settings > Profiles > Terminal > Enable mouse reporting and Report mouse wheel events")
}
