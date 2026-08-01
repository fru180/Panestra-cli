package tmux

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Client struct{ Binary string }

func New() Client { return Client{Binary: "tmux"} }

func (c Client) run(args ...string) (string, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(c.Binary, args...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			return "", fmt.Errorf("%w: %s", err, detail)
		}
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func (c Client) PaneWidth(pane string) (int, error) {
	s, err := c.run("display-message", "-p", "-t", pane, "#{pane_width}")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(s)
}

func (c Client) CurrentPane() (string, error) {
	return c.run("display-message", "-p", "#{pane_id}")
}

func (c Client) SetPrompt(pane, value string) error {
	if pane == "" {
		return fmt.Errorf("tmux pane is empty")
	}
	_, err := c.run("set-option", "-p", "-t", pane, "@panestra_cli_prompt", value)
	return err
}

func (c Client) SetPrefix(pane, value string) error {
	if pane == "" {
		return fmt.Errorf("tmux pane is empty")
	}
	_, err := c.run("set-option", "-p", "-t", pane, "@panestra_cli_prefix", value)
	return err
}

func (c Client) SetWaiting(pane string, show bool) error {
	value := "0"
	if show {
		value = "1"
	}
	_, err := c.run("set-option", "-p", "-t", pane, "@panestra_cli_show_waiting", value)
	return err
}

func (c Client) ClearPrompt(pane string) error {
	_, err := c.run("set-option", "-p", "-u", "-t", pane, "@panestra_cli_prompt")
	return err
}
