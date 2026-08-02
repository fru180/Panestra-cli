package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
)

func Home() string {
	if v := os.Getenv("PANESTRA_CLI_HOME"); v != "" {
		return v
	}
	h, _ := os.UserHomeDir()
	return h
}

func Path() string    { return filepath.Join(Home(), ".config", "panestra-cli", "config.toml") }
func DataDir() string { return filepath.Join(Home(), ".local", "share", "panestra-cli") }
func ShimDir() string { return filepath.Join(DataDir(), "shims") }

func Load() (Config, error) {
	defaults := Default()
	c := defaults
	if _, err := toml.DecodeFile(Path(), &c); err != nil {
		safe := defaults
		safe.Enabled = false
		return safe, fmt.Errorf("load config %s: %w", Path(), err)
	}
	if c.MaxWidth <= 0 {
		c.MaxWidth = defaults.MaxWidth
	}
	if os.Getenv("PANESTRA_CLI_DISABLE") != "" {
		c.Enabled = false
	}
	if v := os.Getenv("PANESTRA_CLI_MAX_WIDTH"); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			c.MaxWidth = n
		}
	}
	if v, ok := os.LookupEnv("PANESTRA_CLI_PREFIX"); ok {
		c.Prefix = v
	}
	return c, nil
}

func Save(c Config) error {
	if err := os.MkdirAll(filepath.Dir(Path()), 0700); err != nil {
		return err
	}
	var content bytes.Buffer
	if err := toml.NewEncoder(&content).Encode(c); err != nil {
		return err
	}
	return os.WriteFile(Path(), content.Bytes(), 0600)
}
