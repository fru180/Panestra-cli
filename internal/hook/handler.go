package hook

import (
	"io"
	"os"

	"github.com/fru180/Panestra-cli/internal/config"
	"github.com/fru180/Panestra-cli/internal/prompt"
	ptmux "github.com/fru180/Panestra-cli/internal/tmux"
)

func Handle(r io.Reader) {
	data, err := io.ReadAll(io.LimitReader(r, 4<<20))
	if err != nil {
		return
	}
	e, err := Parse(data)
	if err != nil {
		debug(err)
		return
	}
	text := prompt.Normalize(e.Prompt)
	if text == "" {
		return
	}
	cfg, err := config.Load()
	if err != nil {
		debug(err)
		return
	}
	if !cfg.Enabled {
		return
	}
	pane := os.Getenv("TMUX_PANE")
	if pane == "" {
		return
	}
	t := ptmux.New()
	active, err := t.IsActive(pane)
	if err != nil {
		debug(err)
		return
	}
	if !active {
		return
	}
	prefix := prompt.DisplayPrefix(cfg.Prefix)
	max := cfg.MaxWidth
	if width, e := t.PaneWidth(pane); e == nil {
		available := width - prompt.Width(prefix) - 2
		if available < max {
			max = available
		}
	}
	if max < 1 {
		max = 1
	}
	text = prompt.Truncate(text, max)
	if err := t.SetPrefix(pane, prefix); err != nil {
		debug(err)
	}
	if err := t.SetPrompt(pane, text); err != nil {
		debug(err)
	}
}

func debug(err error) {
	if os.Getenv("PANESTRA_CLI_DEBUG") != "" {
		_, _ = os.Stderr.WriteString("panestra: " + err.Error() + "\n")
	}
}
