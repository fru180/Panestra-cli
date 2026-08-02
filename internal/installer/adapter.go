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
	changes, err := prepareAdapters(binary, agents)
	if err != nil {
		return err
	}
	return applyFileChanges(changes)
}

func prepareAdapters(binary string, agents []string) ([]fileChange, error) {
	script := "#!/bin/sh\nexec " + shellQuote(binary) + " hook\n"
	changes := []fileChange{{path: hookScriptPath(), data: []byte(script), mode: 0700}}
	for _, agent := range []string{"codex", "claude"} {
		var path string
		if agent == "codex" {
			path = codexHooksPath()
		} else {
			path = claudeSettingsPath()
		}
		change, ok, err := prepareHookChange(path, !hasAgent(agents, agent))
		if err != nil {
			return nil, fmt.Errorf("configure %s hook: %w", agent, err)
		}
		if ok {
			changes = append(changes, change)
		}
	}
	return changes, nil
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
	if err != nil {
		return false
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		return false
	}
	hooks, ok := root["hooks"].(map[string]any)
	if !ok {
		return false
	}
	groups, ok := hooks["UserPromptSubmit"].([]any)
	if !ok {
		return false
	}
	for _, raw := range groups {
		group, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		handlers, ok := group["hooks"].([]any)
		if !ok {
			continue
		}
		for _, rawHandler := range handlers {
			handler, ok := rawHandler.(map[string]any)
			if ok && managedHook(handler) {
				return true
			}
		}
	}
	return false
}

func mergeHook(path string, remove bool) error {
	change, ok, err := prepareHookChange(path, remove)
	if err != nil || !ok {
		return err
	}
	return applyFileChanges([]fileChange{change})
}

func prepareHookChange(path string, remove bool) (fileChange, bool, error) {
	root := map[string]any{}
	b, err := os.ReadFile(path)
	if err == nil && len(b) > 0 {
		var value any
		if err := json.Unmarshal(b, &value); err != nil {
			return fileChange{}, false, fmt.Errorf("invalid JSON in %s: %w", path, err)
		}
		var ok bool
		root, ok = value.(map[string]any)
		if !ok {
			return fileChange{}, false, fmt.Errorf("invalid JSON in %s: root must be an object", path)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return fileChange{}, false, err
	}

	hooks := map[string]any{}
	if raw, exists := root["hooks"]; exists {
		var ok bool
		hooks, ok = raw.(map[string]any)
		if !ok {
			return fileChange{}, false, fmt.Errorf("invalid JSON in %s: hooks must be an object", path)
		}
	} else if remove {
		return fileChange{}, false, nil
	}

	groups := []any{}
	if raw, exists := hooks["UserPromptSubmit"]; exists {
		var ok bool
		groups, ok = raw.([]any)
		if !ok {
			return fileChange{}, false, fmt.Errorf("invalid JSON in %s: hooks.UserPromptSubmit must be an array", path)
		}
	} else if remove {
		return fileChange{}, false, nil
	}

	clean := make([]any, 0, len(groups)+1)
	changed := false
	for _, raw := range groups {
		group, ok := raw.(map[string]any)
		if !ok {
			clean = append(clean, raw)
			continue
		}
		handlers, ok := group["hooks"].([]any)
		if !ok {
			clean = append(clean, raw)
			continue
		}
		kept := make([]any, 0, len(handlers))
		groupChanged := false
		for _, hr := range handlers {
			h, ok := hr.(map[string]any)
			if !ok {
				kept = append(kept, hr)
				continue
			}
			if !managedHook(h) {
				kept = append(kept, hr)
			} else {
				changed = true
				groupChanged = true
			}
		}
		if !groupChanged {
			clean = append(clean, raw)
			continue
		}
		if len(kept) > 0 {
			copy := cloneObject(group)
			copy["hooks"] = kept
			clean = append(clean, copy)
		} else if len(group) > 1 {
			copy := cloneObject(group)
			copy["hooks"] = []any{}
			clean = append(clean, copy)
		}
	}
	if !remove {
		clean = append(clean, map[string]any{"hooks": []any{map[string]any{"type": "command", "command": shellQuote(hookScriptPath()), "timeout": 1}}})
		changed = true
	}
	if remove && !changed {
		return fileChange{}, false, nil
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
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fileChange{}, false, err
	}
	out = append(out, '\n')
	return fileChange{path: path, data: out, mode: 0600}, true, nil
}

func cloneObject(value map[string]any) map[string]any {
	copy := make(map[string]any, len(value))
	for key, item := range value {
		copy[key] = item
	}
	return copy
}

func managedHook(handler map[string]any) bool {
	typeName, typeOK := handler["type"].(string)
	command, commandOK := handler["command"].(string)
	return typeOK && typeName == "command" && commandOK && command == shellQuote(hookScriptPath())
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
