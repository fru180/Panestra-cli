package main

import (
	"fmt"
	"os"

	"github.com/fru180/Panestra-cli/internal/config"
	"github.com/fru180/Panestra-cli/internal/hook"
	"github.com/fru180/Panestra-cli/internal/installer"
	"github.com/fru180/Panestra-cli/internal/launcher"
	ptmux "github.com/fru180/Panestra-cli/internal/tmux"
)

func main() { os.Exit(run()) }

func run() int {
	if len(os.Args) < 2 {
		usage()
		return 2
	}
	switch os.Args[1] {
	case "setup":
		if err := installer.Setup(); err != nil {
			fail(err)
			return 1
		}
		return 0
	case "doctor":
		if !installer.Doctor() {
			return 1
		}
		return 0
	case "uninstall":
		if err := installer.Uninstall(); err != nil {
			fail(err)
			return 1
		}
		fmt.Println("Panestra CLI has been uninstalled.")
		return 0
	case "enable", "disable":
		c, err := config.Load()
		if err != nil {
			fail(fmt.Errorf("%w; fix %s, then rerun panestra %s", err, config.Path(), os.Args[1]))
			return 1
		}
		c.Enabled = os.Args[1] == "enable"
		if err := config.Save(c); err != nil {
			fail(err)
			return 1
		}
		fmt.Println("Panestra CLI", os.Args[1]+"d.")
		return 0
	case "clear":
		pane := os.Getenv("TMUX_PANE")
		if pane == "" && os.Getenv("TMUX") != "" {
			pane, _ = ptmux.New().CurrentPane()
		}
		if pane == "" {
			fail(fmt.Errorf("not running in a tmux pane"))
			return 1
		}
		if err := ptmux.New().ClearPrompt(pane); err != nil {
			fail(err)
			return 1
		}
		return 0
	case "hook":
		hook.Handle(os.Stdin)
		return 0
	case "launch":
		if len(os.Args) < 3 {
			fail(fmt.Errorf("agent is required"))
			return 2
		}
		return launcher.Launch(os.Args[2], os.Args[3:])
	case "_exec-status":
		if len(os.Args) != 5 {
			return 126
		}
		return launcher.ExecStatus(os.Args[2], os.Args[3], os.Args[4])
	case "version", "--version", "-V":
		fmt.Println("panestra", config.Version)
		return 0
	case "help", "--help", "-h":
		usage()
		return 0
	default:
		fail(fmt.Errorf("unknown command %q", os.Args[1]))
		usage()
		return 2
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, "panestra:", err) }
func usage() {
	fmt.Fprintln(os.Stderr, "Usage: panestra <setup|doctor|uninstall|enable|disable|clear|hook|launch|version>")
}
