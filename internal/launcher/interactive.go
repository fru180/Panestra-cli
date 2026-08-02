package launcher

import "strings"

func Interactive(agent string, args []string, stdinTTY, stdoutTTY bool) bool {
	if !stdinTTY || !stdoutTTY {
		return false
	}
	for _, a := range args {
		switch a {
		case "-h", "--help", "-V", "--version":
			return false
		}
	}
	if len(args) == 0 {
		return true
	}
	if agent == "codex" {
		switch args[0] {
		case "exec", "review", "completion", "mcp", "app-server", "cloud", "login", "logout":
			return false
		}
	}
	if agent == "claude" {
		for _, a := range args {
			if a == "-p" || a == "--print" || strings.HasPrefix(a, "--print=") {
				return false
			}
		}
	}
	return true
}
