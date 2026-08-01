package tmux

import (
	"errors"
	"fmt"
	"strconv"
)

const borderFormat = `#{?#{@panestra_cli_active},#{?#{@panestra_cli_prompt}, #{@panestra_cli_prefix}#{@panestra_cli_prompt} ,#{?#{@panestra_cli_show_waiting}, Panestra CLI — Waiting for prompt ,}},#{@panestra_cli_previous_border_format}}`

func (c Client) Acquire(pane, prefix string, showWaiting bool) error {
	if pane == "" {
		return fmt.Errorf("tmux pane is empty")
	}
	refText, _ := c.run("show-options", "-wqv", "-t", pane, "@panestra_cli_refcount")
	ref, _ := strconv.Atoi(refText)
	if ref == 0 {
		status, err := c.run("show-options", "-wAv", "-t", pane, "pane-border-status")
		if err != nil {
			return err
		}
		format, err := c.run("show-options", "-wAv", "-t", pane, "pane-border-format")
		if err != nil {
			return err
		}
		if _, err = c.run("set-option", "-w", "-t", pane, "@panestra_cli_previous_border_status", status); err != nil {
			return err
		}
		if _, err = c.run("set-option", "-w", "-t", pane, "@panestra_cli_previous_border_format", format); err != nil {
			return err
		}
		if _, err = c.run("set-option", "-w", "-t", pane, "pane-border-status", "top"); err != nil {
			return err
		}
		if _, err = c.run("set-option", "-w", "-t", pane, "pane-border-format", borderFormat); err != nil {
			return err
		}
	}
	if _, err := c.run("set-option", "-w", "-t", pane, "@panestra_cli_refcount", strconv.Itoa(ref+1)); err != nil {
		return err
	}
	if _, err := c.run("set-option", "-p", "-t", pane, "@panestra_cli_active", "1"); err != nil {
		return err
	}
	if _, err := c.run("set-option", "-p", "-t", pane, "@panestra_cli_prefix", prefix); err != nil {
		return err
	}
	return c.SetWaiting(pane, showWaiting)
}

func (c Client) Release(pane string) error {
	var errs []error
	if _, err := c.run("set-option", "-p", "-u", "-t", pane, "@panestra_cli_active"); err != nil {
		errs = append(errs, err)
	}
	if _, err := c.run("set-option", "-p", "-u", "-t", pane, "@panestra_cli_prefix"); err != nil {
		errs = append(errs, err)
	}
	if _, err := c.run("set-option", "-p", "-u", "-t", pane, "@panestra_cli_show_waiting"); err != nil {
		errs = append(errs, err)
	}
	if err := c.ClearPrompt(pane); err != nil {
		errs = append(errs, err)
	}
	refText, _ := c.run("show-options", "-wqv", "-t", pane, "@panestra_cli_refcount")
	ref, _ := strconv.Atoi(refText)
	if ref > 1 {
		_, err := c.run("set-option", "-w", "-t", pane, "@panestra_cli_refcount", strconv.Itoa(ref-1))
		if err != nil {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	}
	status, _ := c.run("show-options", "-wqv", "-t", pane, "@panestra_cli_previous_border_status")
	format, _ := c.run("show-options", "-wqv", "-t", pane, "@panestra_cli_previous_border_format")
	if status != "" {
		if _, err := c.run("set-option", "-w", "-t", pane, "pane-border-status", status); err != nil {
			errs = append(errs, err)
		}
	}
	if format != "" {
		if _, err := c.run("set-option", "-w", "-t", pane, "pane-border-format", format); err != nil {
			errs = append(errs, err)
		}
	}
	for _, key := range []string{"@panestra_cli_refcount", "@panestra_cli_previous_border_status", "@panestra_cli_previous_border_format"} {
		if _, err := c.run("set-option", "-wu", "-t", pane, key); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
