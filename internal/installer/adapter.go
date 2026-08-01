package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fru180/Panestra-cli/internal/config"
)

func hookScriptPath() string { return filepath.Join(config.DataDir(), "adapters", "panestra-hook") }
func codexHooksPath() string { return filepath.Join(config.Home(), ".codex", "hooks.json") }
func claudeSettingsPath() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "settings.json")
	}
	return filepath.Join(config.Home(), ".claude", "settings.json")
}

func installAdapters(binary string, agents []string) error {
	script := "#!/bin/sh\nexec " + shellQuote(binary) + " hook\n"
	if err := atomicWrite(hookScriptPath(), []byte(script), 0700); err != nil {
		return err
	}
	for _, agent := range []string{"codex", "claude"} {
		var path string
		if agent == "codex" {
			path = codexHooksPath()
		} else {
			path = claudeSettingsPath()
		}
		if err := mergeHook(path, !hasAgent(agents, agent)); err != nil {
			return fmt.Errorf("configure %s hook: %w", agent, err)
		}
	}
	return nil
}

func hasAgent(agents []string, target string) bool {
	for _, agent := range agents {
		if agent == target {
			return true
		}
	}
	return false
}

func uninstallAdapters() error {
	var errs []string
	for _, path := range []string{codexHooksPath(), claudeSettingsPath()} {
		if err := mergeHook(path, true); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func adapterInstalled(path string) bool {
	b, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(b), filepath.Join("panestra-cli", "adapters", "panestra-hook"))
}

func mergeHook(path string, remove bool) error {
	root := map[string]any{}
	b, err := os.ReadFile(path)
	if err == nil && len(b) > 0 {
		if err := json.Unmarshal(b, &root); err != nil {
			return fmt.Errorf("invalid JSON in %s: %w", path, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	groups, _ := hooks["UserPromptSubmit"].([]any)
	clean := make([]any, 0, len(groups)+1)
	for _, raw := range groups {
		group, ok := raw.(map[string]any)
		if !ok {
			clean = append(clean, raw)
			continue
		}
		handlers, _ := group["hooks"].([]any)
		kept := make([]any, 0, len(handlers))
		for _, hr := range handlers {
			h, ok := hr.(map[string]any)
			if !ok {
				kept = append(kept, hr)
				continue
			}
			cmd, _ := h["command"].(string)
			if !strings.Contains(cmd, filepath.Join("panestra-cli", "adapters", "panestra-hook")) {
				kept = append(kept, hr)
			}
		}
		if len(kept) > 0 {
			group["hooks"] = kept
			clean = append(clean, group)
		}
	}
	if !remove {
		clean = append(clean, map[string]any{"hooks": []any{map[string]any{"type": "command", "command": shellQuote(hookScriptPath()), "timeout": 1}}})
	}
	if len(clean) == 0 {
		delete(hooks, "UserPromptSubmit")
	} else {
		hooks["UserPromptSubmit"] = clean
	}
	if len(hooks) == 0 {
		delete(root, "hooks")
	} else {
		root["hooks"] = hooks
	}
	if remove && os.IsNotExist(err) {
		return nil
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return atomicWrite(path, out, 0600)
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
