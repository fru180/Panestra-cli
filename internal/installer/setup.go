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

func Setup() error {
	if runtime.GOOS != "darwin" && os.Getenv("PANESTRA_CLI_ALLOW_UNSUPPORTED") == "" {
		return fmt.Errorf("macOS is required (set PANESTRA_CLI_ALLOW_UNSUPPORTED=1 for development)")
	}
	if shell := os.Getenv("SHELL"); filepath.Base(shell) != "zsh" && os.Getenv("PANESTRA_CLI_ALLOW_UNSUPPORTED") == "" {
		if shell == "" {
			shell = "unknown"
		}
		return fmt.Errorf("zsh is required; current shell is %s", shell)
	}
	binary, err := os.Executable()
	if err != nil {
		return err
	}
	binary, _ = filepath.EvalSymlinks(binary)
	agents := []string{}
	for _, name := range []string{"codex", "claude"} {
		if _, err := launcher.Resolve(name); err == nil {
			agents = append(agents, name)
		}
	}
	if len(agents) == 0 {
		return fmt.Errorf("neither Codex CLI nor Claude Code was found in PATH")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		fmt.Fprintln(os.Stderr, "warning: tmux was not found; install it with: brew install tmux")
	}
	if err := os.MkdirAll(config.ShimDir(), 0700); err != nil {
		return err
	}
	for _, agent := range agents {
		shim := "#!/bin/sh\nexec " + shellQuote(binary) + " launch " + agent + " \"$@\"\n"
		if err := atomicWrite(filepath.Join(config.ShimDir(), agent), []byte(shim), 0700); err != nil {
			return err
		}
	}
	for _, candidate := range []string{"codex", "claude"} {
		if !hasAgent(agents, candidate) {
			if err := os.Remove(filepath.Join(config.ShimDir(), candidate)); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	if err := addPathBlock(); err != nil {
		return err
	}
	if _, err := os.Stat(config.Path()); os.IsNotExist(err) {
		if err := config.Save(config.Default()); err != nil {
			return err
		}
	}
	if err := installAdapters(binary, agents); err != nil {
		return err
	}
	fmt.Println("Panestra CLI displays your latest prompt on screen.")
	fmt.Println("Prompts may be visible during screen sharing or recording.")
	fmt.Println("Setup complete. Run: source ~/.zshrc")
	fmt.Println()
	Doctor()
	return nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".panestra-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func IsShimActive(agent string) bool {
	p, err := exec.LookPath(agent)
	if err != nil {
		return false
	}
	a, _ := filepath.EvalSymlinks(p)
	b, _ := filepath.EvalSymlinks(filepath.Join(config.ShimDir(), agent))
	return a == b || strings.TrimSpace(a) == strings.TrimSpace(b)
}
