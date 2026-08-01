package config

const Version = "0.1.0"

type Config struct {
	Enabled            bool   `toml:"enabled"`
	AutoTmux           bool   `toml:"auto_tmux"`
	Prefix             string `toml:"prefix"`
	MaxWidth           int    `toml:"max_width"`
	ShowWaitingMessage bool   `toml:"show_waiting_message"`
}

func Default() Config {
	return Config{Enabled: true, AutoTmux: true, Prefix: "Task: ", MaxWidth: 120, ShowWaitingMessage: true}
}
