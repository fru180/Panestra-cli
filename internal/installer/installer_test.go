package installer

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fru180/Panestra-cli/internal/config"
)

func TestMergeHookPreservesExistingAndIsIdempotent(t *testing.T) {
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	path := codexHooksPath()
	existing := `{"other":true,"hooks":{"UserPromptSubmit":[{"hooks":[{"type":"command","command":"keep-me"}]}]}}`
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}
	if err := mergeHook(path, false); err != nil {
		t.Fatal(err)
	}
	if err := mergeHook(path, false); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if strings.Count(string(b), "panestra-hook") != 1 {
		t.Fatalf("hook duplicated: %s", b)
	}
	if !strings.Contains(string(b), "keep-me") {
		t.Fatalf("existing hook lost: %s", b)
	}
	if err := mergeHook(path, true); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(path)
	if strings.Contains(string(b), "panestra-hook") || !strings.Contains(string(b), "keep-me") {
		t.Fatalf("bad removal: %s", b)
	}
}

func TestManagedHookRequiresExactGeneratedCommand(t *testing.T) {
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	exact := shellQuote(hookScriptPath())
	tests := []struct {
		name    string
		handler map[string]any
		want    bool
	}{
		{name: "managed", handler: map[string]any{"type": "command", "command": exact}, want: true},
		{name: "backup", handler: map[string]any{"type": "command", "command": shellQuote(hookScriptPath() + ".backup")}},
		{name: "additional arguments", handler: map[string]any{"type": "command", "command": exact + " --do-not-delete"}},
		{name: "similar path", handler: map[string]any{"type": "command", "command": shellQuote(filepath.Join(config.DataDir(), "other", "panestra-hook"))}},
		{name: "unquoted", handler: map[string]any{"type": "command", "command": hookScriptPath()}},
		{name: "different handler type", handler: map[string]any{"type": "prompt", "command": exact}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := managedHook(tt.handler); got != tt.want {
				t.Fatalf("managedHook() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeHookPreservesSimilarCommandsAndMatcherFields(t *testing.T) {
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	path := codexHooksPath()
	exact := shellQuote(hookScriptPath())
	similar := []string{
		shellQuote(hookScriptPath() + ".backup"),
		exact + " --do-not-delete",
		shellQuote(filepath.Join(config.DataDir(), "other", "panestra-hook")),
		hookScriptPath(),
	}
	root := map[string]any{
		"hooks": map[string]any{
			"UserPromptSubmit": []any{
				map[string]any{
					"matcher": "preserve-me",
					"future":  map[string]any{"enabled": true},
					"hooks":   []any{map[string]any{"type": "command", "command": exact}},
				},
				map[string]any{
					"hooks": []any{
						map[string]any{"type": "command", "command": similar[0]},
						map[string]any{"type": "command", "command": similar[1]},
						map[string]any{"type": "command", "command": similar[2]},
						map[string]any{"type": "command", "command": similar[3]},
					},
				},
			},
		},
	}
	b, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}

	if err := mergeHook(path, false); err != nil {
		t.Fatal(err)
	}
	if err := mergeHook(path, false); err != nil {
		t.Fatal(err)
	}
	if !adapterInstalled(path) {
		t.Fatal("managed hook was not detected")
	}
	commands, groups := readHookCommands(t, path)
	if countValue(commands, exact) != 1 {
		t.Fatalf("managed command count = %d, want 1: %#v", countValue(commands, exact), commands)
	}
	for _, command := range similar {
		if countValue(commands, command) != 1 {
			t.Fatalf("similar command %q was changed: %#v", command, commands)
		}
	}
	assertMatcherGroupPreserved(t, groups)

	if err := mergeHook(path, true); err != nil {
		t.Fatal(err)
	}
	afterFirstUninstall, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := mergeHook(path, true); err != nil {
		t.Fatal(err)
	}
	afterSecondUninstall, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterSecondUninstall, afterFirstUninstall) {
		t.Fatal("repeated uninstall changed hook configuration")
	}
	if adapterInstalled(path) {
		t.Fatal("managed hook remains installed")
	}
	commands, groups = readHookCommands(t, path)
	if countValue(commands, exact) != 0 {
		t.Fatalf("managed command remains: %#v", commands)
	}
	for _, command := range similar {
		if countValue(commands, command) != 1 {
			t.Fatalf("similar command %q was changed by uninstall: %#v", command, commands)
		}
	}
	assertMatcherGroupPreserved(t, groups)
}

func TestAdapterInstalledIgnoresManagedCommandOutsideUserPromptSubmit(t *testing.T) {
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	path := codexHooksPath()
	root := map[string]any{
		"note": shellQuote(hookScriptPath()),
		"hooks": map[string]any{
			"OtherEvent": []any{map[string]any{
				"hooks": []any{map[string]any{"type": "command", "command": shellQuote(hookScriptPath())}},
			}},
		},
	}
	b, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	if adapterInstalled(path) {
		t.Fatal("command outside UserPromptSubmit was detected as installed")
	}
}

func TestMergeHookRejectsInvalidJSONShapes(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "malformed", content: `{`, want: "invalid JSON"},
		{name: "null root", content: `null`, want: "root must be an object"},
		{name: "array root", content: `[]`, want: "root must be an object"},
		{name: "string root", content: `"value"`, want: "root must be an object"},
		{name: "null hooks", content: `{"hooks":null}`, want: "hooks must be an object"},
		{name: "array hooks", content: `{"hooks":[]}`, want: "hooks must be an object"},
		{name: "object event", content: `{"hooks":{"UserPromptSubmit":{}}}`, want: "hooks.UserPromptSubmit must be an array"},
		{name: "string event", content: `{"hooks":{"UserPromptSubmit":"value"}}`, want: "hooks.UserPromptSubmit must be an array"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			if err := os.WriteFile(path, []byte(tt.content), 0600); err != nil {
				t.Fatal(err)
			}
			if err := mergeHook(path, false); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got error %v, want containing %q", err, tt.want)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.content {
				t.Fatalf("invalid configuration changed: got %q, want %q", got, tt.content)
			}
		})
	}
}

func TestMergeHookPreservesUnknownValidData(t *testing.T) {
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	path := codexHooksPath()
	existing := `{
  "other": {"nested": true},
  "hooks": {
    "OtherEvent": {"custom": 1},
    "UserPromptSubmit": [
      "future-group",
      {"matcher": "future", "hooks": "future-shape"},
      {"hooks": []},
      {"hooks": [{"type": "command", "command": "keep-me", "future": true}]}
    ]
  }
}`
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}
	if err := mergeHook(path, false); err != nil {
		t.Fatal(err)
	}
	if err := mergeHook(path, false); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(b), "panestra-hook") != 1 {
		t.Fatalf("hook was not idempotent: %s", b)
	}
	if err := mergeHook(path, true); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got, want any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(existing), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unknown data changed:\n got: %s\nwant: %s", b, existing)
	}
}

func TestPathBlockIdempotent(t *testing.T) {
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	if err := os.WriteFile(zshrcPath(), []byte("export KEEP=1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := addPathBlock(); err != nil {
		t.Fatal(err)
	}
	if err := addPathBlock(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(zshrcPath())
	if strings.Count(string(b), beginMarker) != 1 || !strings.Contains(string(b), "export KEEP=1") {
		t.Fatalf("bad zshrc: %s", b)
	}
	if info, _ := os.Stat(zshrcPath()); info.Mode().Perm() != 0644 {
		t.Fatalf("permissions changed to %o", info.Mode().Perm())
	}
	if err := removePathBlock(); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(zshrcPath())
	if strings.Contains(string(b), beginMarker) || !strings.Contains(string(b), "export KEEP=1") {
		t.Fatalf("bad cleanup: %s", b)
	}
}

func TestSetupAndUninstallPreserveSymlinkedUserConfiguration(t *testing.T) {
	home, fakeBin := t.TempDir(), t.TempDir()
	t.Setenv("PANESTRA_CLI_HOME", home)
	t.Setenv("PANESTRA_CLI_ALLOW_UNSUPPORTED", "1")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("SHELL", "/bin/zsh")
	for _, agent := range []string{"codex", "claude"} {
		writeExecutable(t, filepath.Join(fakeBin, agent), "#!/bin/sh\nexit 0\n")
	}
	t.Setenv("PATH", fakeBin)

	dotfiles := filepath.Join(home, "dotfiles")
	for _, dir := range []string{dotfiles, filepath.Dir(codexHooksPath()), filepath.Dir(claudeSettingsPath())} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	targets := []struct {
		path       string
		link       string
		linkTarget string
		content    string
		mode       os.FileMode
	}{
		{
			path:       filepath.Join(dotfiles, "zshrc"),
			link:       zshrcPath(),
			linkTarget: filepath.Join("dotfiles", "zshrc"),
			content:    "export KEEP=1\n",
			mode:       0644,
		},
		{
			path:       filepath.Join(dotfiles, "codex-hooks.json"),
			link:       codexHooksPath(),
			linkTarget: filepath.Join("..", "dotfiles", "codex-hooks.json"),
			content:    "{\"codexKeep\":true}\n",
			mode:       0640,
		},
		{
			path:       filepath.Join(dotfiles, "claude-settings.json"),
			link:       claudeSettingsPath(),
			linkTarget: filepath.Join(dotfiles, "claude-settings.json"),
			content:    "{\"claudeKeep\":true}\n",
			mode:       0604,
		},
	}
	for _, target := range targets {
		if err := os.WriteFile(target.path, []byte(target.content), target.mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target.linkTarget, target.link); err != nil {
			t.Fatal(err)
		}
	}

	if err := Setup(); err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		assertSymlink(t, target.link, target.linkTarget)
		info, err := os.Stat(target.path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != target.mode {
			t.Fatalf("mode for %s = %o, want %o", target.path, info.Mode().Perm(), target.mode)
		}
	}
	zshrc, err := os.ReadFile(targets[0].path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(zshrc), "export KEEP=1") || !strings.Contains(string(zshrc), beginMarker) {
		t.Fatalf("symlinked zshrc was not updated correctly: %s", zshrc)
	}
	for _, target := range targets[1:] {
		if !adapterInstalled(target.link) {
			t.Fatalf("managed hook was not installed through %s", target.link)
		}
	}

	if err := Uninstall(); err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		assertSymlink(t, target.link, target.linkTarget)
	}
	zshrc, err = os.ReadFile(targets[0].path)
	if err != nil {
		t.Fatal(err)
	}
	if string(zshrc) != "export KEEP=1\n" {
		t.Fatalf("symlinked zshrc was not cleaned correctly: %q", zshrc)
	}
	for _, target := range targets[1:] {
		content, err := os.ReadFile(target.path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "panestra-hook") || !strings.Contains(string(content), "Keep") {
			t.Fatalf("symlinked agent settings were not cleaned correctly: %s", content)
		}
	}
}

func TestAtomicWriteRejectsInvalidSymlinkTargets(t *testing.T) {
	tests := []struct {
		name       string
		linkTarget func(string) string
		want       string
	}{
		{
			name: "dangling",
			linkTarget: func(root string) string {
				return filepath.Join(root, "missing")
			},
			want: "resolve symlink",
		},
		{
			name: "directory",
			linkTarget: func(root string) string {
				return root
			},
			want: "not a regular file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			link := filepath.Join(root, "settings.json")
			target := tt.linkTarget(root)
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			err := atomicWrite(link, []byte("changed\n"), 0600)
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), link) {
				t.Fatalf("got error %v, want containing %q and %q", err, tt.want, link)
			}
			assertSymlink(t, link, target)
		})
	}
}

func TestAtomicWriteReportsSymlinkTargetPermissionError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write to read-only directories")
	}
	root := t.TempDir()
	targetDir := filepath.Join(root, "target")
	if err := os.Mkdir(targetDir, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(targetDir, "zshrc")
	if err := os.WriteFile(target, []byte("before\n"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "zshrc-link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(targetDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(targetDir, 0700) })

	err := atomicWrite(link, []byte("after\n"), 0600)
	if err == nil || !strings.Contains(err.Error(), "create temporary file") || !strings.Contains(err.Error(), link) {
		t.Fatalf("got error %v, want a contextual permission error", err)
	}
	assertSymlink(t, link, target)
	content, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(content) != "before\n" {
		t.Fatalf("permission failure changed target: %q", content)
	}
}

func TestInstallAdaptersWritesValidAgentFiles(t *testing.T) {
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	if err := installAdapters("/opt/bin/panestra", []string{"codex", "claude"}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{codexHooksPath(), claudeSettingsPath()} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var root map[string]any
		if json.Unmarshal(b, &root) != nil {
			t.Fatalf("invalid JSON: %s", b)
		}
		if !strings.Contains(string(b), "UserPromptSubmit") {
			t.Fatalf("missing event: %s", b)
		}
	}
	info, err := os.Stat(hookScriptPath())
	if err != nil || info.Mode()&0100 == 0 {
		t.Fatal("hook script is not executable")
	}
}

func TestUninstallRemovesOnlyManagedFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PANESTRA_CLI_HOME", home)
	if err := config.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	if err := installAdapters("/opt/bin/panestra", []string{"codex"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config.DataDir(), "keep-until-uninstall"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(zshrcPath(), []byte("export KEEP=1\n"+beginMarker+"\nmanaged\n"+endMarker+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(config.DataDir()); !os.IsNotExist(err) {
		t.Fatal("data directory remains")
	}
	b, _ := os.ReadFile(zshrcPath())
	if string(b) != "export KEEP=1\n" {
		t.Fatalf("zshrc damaged: %q", b)
	}
}

func TestSetupAgentCombinations(t *testing.T) {
	for _, agents := range [][]string{{"codex"}, {"claude"}, {"codex", "claude"}} {
		name := strings.Join(agents, "-")
		t.Run(name, func(t *testing.T) {
			home, fakeBin := t.TempDir(), t.TempDir()
			t.Setenv("PANESTRA_CLI_HOME", home)
			t.Setenv("PANESTRA_CLI_ALLOW_UNSUPPORTED", "1")
			t.Setenv("CLAUDE_CONFIG_DIR", "")
			t.Setenv("SHELL", "/bin/zsh")
			for _, agent := range agents {
				path := filepath.Join(fakeBin, agent)
				if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", fakeBin)
			if err := Setup(); err != nil {
				t.Fatal(err)
			}
			beforeSecondSetup := snapshotTree(t, home)
			shimPath := config.ShimDir() + string(os.PathListSeparator) + fakeBin
			t.Setenv("PATH", shimPath)
			if err := Setup(); err != nil {
				t.Fatal(err)
			}
			if afterSecondSetup := snapshotTree(t, home); !reflect.DeepEqual(afterSecondSetup, beforeSecondSetup) {
				t.Fatalf("second setup changed filesystem:\n got: %#v\nwant: %#v", afterSecondSetup, beforeSecondSetup)
			}
			zshrc, err := os.ReadFile(zshrcPath())
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(string(zshrc), beginMarker) != 1 {
				t.Fatalf("PATH block duplicated: %s", zshrc)
			}
			for _, candidate := range []string{"codex", "claude"} {
				installed := false
				for _, agent := range agents {
					installed = installed || candidate == agent
				}
				_, err := os.Stat(filepath.Join(config.ShimDir(), candidate))
				if installed && err != nil {
					t.Fatalf("%s shim missing: %v", candidate, err)
				}
				if !installed && !os.IsNotExist(err) {
					t.Fatalf("unexpected %s shim", candidate)
				}
			}
			if contains(agents, "codex") && !adapterInstalled(codexHooksPath()) {
				t.Fatal("Codex adapter missing")
			}
			if contains(agents, "claude") && !adapterInstalled(claudeSettingsPath()) {
				t.Fatal("Claude adapter missing")
			}
		})
	}
}

func TestShimsAndHookSurviveLauncherUpgrade(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PANESTRA_CLI_HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	root := filepath.Join(t.TempDir(), "prefix with space and 'quote")
	binDir := filepath.Join(root, "bin")
	oldVersion := filepath.Join(root, "Cellar", "panestra-cli", "1.0.0")
	newVersion := filepath.Join(root, "Cellar", "panestra-cli", "1.1.0")
	oldLauncher := filepath.Join(oldVersion, "bin", "panestra")
	newLauncher := filepath.Join(newVersion, "bin", "panestra")
	stableLauncher := filepath.Join(binDir, "panestra")
	for _, dir := range []string{binDir, filepath.Dir(oldLauncher), filepath.Dir(newLauncher)} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeExecutable(t, oldLauncher, "#!/bin/sh\nprintf 'old launcher must not run\\n' >&2\nexit 99\n")
	writeExecutable(t, newLauncher, "#!/bin/sh\nprintf '%s\\n' \"$@\"\n")
	if err := os.Symlink(oldLauncher, stableLauncher); err != nil {
		t.Fatal(err)
	}

	hijackDir := t.TempDir()
	writeExecutable(t, filepath.Join(hijackDir, "panestra"), "#!/bin/sh\nexit 98\n")
	pathEnv := hijackDir + string(os.PathListSeparator) + binDir
	launcher := stableLauncherPath(oldLauncher, pathEnv)
	if launcher != stableLauncher {
		t.Fatalf("launcher = %q, want stable symlink %q", launcher, stableLauncher)
	}
	if err := installShims(launcher, []string{"codex", "claude"}); err != nil {
		t.Fatal(err)
	}
	if err := installAdapters(launcher, []string{"codex", "claude"}); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(stableLauncher); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(newLauncher, stableLauncher); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(oldVersion); err != nil {
		t.Fatal(err)
	}

	for _, agent := range []string{"codex", "claude"} {
		output, err := exec.Command(filepath.Join(config.ShimDir(), agent), "argument with spaces", "quote'argument").CombinedOutput()
		if err != nil {
			t.Fatalf("%s shim failed after upgrade: %v: %s", agent, err, output)
		}
		want := "launch\n" + agent + "\nargument with spaces\nquote'argument\n"
		if string(output) != want {
			t.Fatalf("%s shim output = %q, want %q", agent, output, want)
		}
	}
	output, err := exec.Command(hookScriptPath()).CombinedOutput()
	if err != nil {
		t.Fatalf("hook failed after upgrade: %v: %s", err, output)
	}
	if string(output) != "hook\n" {
		t.Fatalf("hook output = %q, want %q", output, "hook\\n")
	}
}

func TestStableLauncherPathSupportsCustomPrefix(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PANESTRA_CLI_HOME", home)
	launcher := filepath.Join(t.TempDir(), "custom prefix", "bin", "panestra")
	if err := os.MkdirAll(filepath.Dir(launcher), 0700); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, launcher, "#!/bin/sh\nprintf '%s\\n' \"$@\"\n")
	if got := stableLauncherPath(launcher, filepath.Dir(launcher)); got != launcher {
		t.Fatalf("launcher = %q, want custom prefix path %q", got, launcher)
	}
	if err := installShims(launcher, []string{"codex"}); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(filepath.Join(config.ShimDir(), "codex"), "custom argument").CombinedOutput()
	if err != nil {
		t.Fatalf("custom prefix shim failed: %v: %s", err, output)
	}
	if string(output) != "launch\ncodex\ncustom argument\n" {
		t.Fatalf("shim output = %q", output)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0700); err != nil {
		t.Fatal(err)
	}
}

func TestSetupValidationFailureLeavesFilesystemUnchanged(t *testing.T) {
	home, fakeBin := t.TempDir(), t.TempDir()
	t.Setenv("PANESTRA_CLI_HOME", home)
	t.Setenv("PANESTRA_CLI_ALLOW_UNSUPPORTED", "1")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("SHELL", "/bin/zsh")
	for _, agent := range []string{"codex", "claude"} {
		if err := os.WriteFile(filepath.Join(fakeBin, agent), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", fakeBin)

	files := []struct {
		path    string
		content string
		mode    os.FileMode
	}{
		{path: filepath.Join(config.ShimDir(), "codex"), content: "old codex shim\n", mode: 0711},
		{path: filepath.Join(config.ShimDir(), "claude"), content: "old claude shim\n", mode: 0701},
		{path: zshrcPath(), content: "export KEEP=1\n", mode: 0644},
		{path: config.Path(), content: "enabled = false\n", mode: 0600},
		{path: hookScriptPath(), content: "old hook script\n", mode: 0750},
		{path: codexHooksPath(), content: `{"hooks":{"UserPromptSubmit":[{"hooks":[{"command":"keep-me"}]}]}}`, mode: 0640},
		{path: claudeSettingsPath(), content: `{`, mode: 0600},
	}
	for _, file := range files {
		if err := os.MkdirAll(filepath.Dir(file.path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file.path, []byte(file.content), file.mode); err != nil {
			t.Fatal(err)
		}
	}

	before := snapshotTree(t, home)
	err := Setup()
	if err == nil || !strings.Contains(err.Error(), "configure claude hook") {
		t.Fatalf("got error %v, want Claude configuration error", err)
	}
	if after := snapshotTree(t, home); !reflect.DeepEqual(after, before) {
		t.Fatalf("failed setup changed filesystem:\n got: %#v\nwant: %#v", after, before)
	}
}

func TestApplyFileChangesRollsBackWritesDeletesAndDirectories(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "existing")
	stale := filepath.Join(root, "stale")
	staleTarget := filepath.Join(root, "targets", "stale-target")
	linkedTarget := filepath.Join(root, "targets", "linked-target")
	linked := filepath.Join(root, "linked")
	created := filepath.Join(root, "new", "nested", "created")
	for _, file := range []struct {
		path    string
		content string
		mode    os.FileMode
	}{
		{path: existing, content: "before\n", mode: 0640},
		{path: staleTarget, content: "restore me\n", mode: 0604},
		{path: linkedTarget, content: "linked before\n", mode: 0644},
	} {
		if err := os.MkdirAll(filepath.Dir(file.path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file.path, []byte(file.content), file.mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join("targets", "stale-target"), stale); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("targets", "linked-target"), linked); err != nil {
		t.Fatal(err)
	}
	before := snapshotTree(t, root)
	wantErr := errors.New("injected commit failure")
	changes := []fileChange{
		{path: existing, data: []byte("after\n"), mode: 0600},
		{path: stale, remove: true},
		{path: linked, data: []byte("linked after\n"), mode: 0600},
		{path: created, data: []byte("new\n"), mode: 0600},
		{path: filepath.Join(root, "never-written"), data: []byte("x"), mode: 0600},
	}
	applied := 0
	err := applyFileChangesWith(changes, func(change fileChange) error {
		applied++
		if applied == len(changes) {
			return wantErr
		}
		return applyFileChange(change)
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
	if after := snapshotTree(t, root); !reflect.DeepEqual(after, before) {
		t.Fatalf("rollback did not restore filesystem:\n got: %#v\nwant: %#v", after, before)
	}
}

func TestApplyFileChangesRejectsDanglingSymlinkBeforeChanges(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "existing")
	if err := os.WriteFile(existing, []byte("before\n"), 0644); err != nil {
		t.Fatal(err)
	}
	dangling := filepath.Join(root, "dangling")
	if err := os.Symlink("missing", dangling); err != nil {
		t.Fatal(err)
	}
	before := snapshotTree(t, root)

	err := applyFileChanges([]fileChange{
		{path: existing, data: []byte("after\n"), mode: 0600},
		{path: dangling, data: []byte("new\n"), mode: 0600},
	})
	if err == nil || !strings.Contains(err.Error(), "resolve symlink") || !strings.Contains(err.Error(), dangling) {
		t.Fatalf("got error %v, want dangling symlink context", err)
	}
	if after := snapshotTree(t, root); !reflect.DeepEqual(after, before) {
		t.Fatalf("dangling symlink validation changed filesystem:\n got: %#v\nwant: %#v", after, before)
	}
}

func assertSymlink(t *testing.T, path, wantTarget string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symlink", path)
	}
	target, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	if target != wantTarget {
		t.Fatalf("symlink target for %s = %q, want %q", path, target, wantTarget)
	}
}

type snapshotEntry struct {
	mode    os.FileMode
	content string
}

func snapshotTree(t *testing.T, root string) map[string]snapshotEntry {
	t.Helper()
	snapshot := map[string]snapshotEntry{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		item := snapshotEntry{mode: info.Mode()}
		if info.Mode().IsRegular() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			item.content = string(content)
		}
		snapshot[relative] = item
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func readHookCommands(t *testing.T, path string) ([]string, []any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatal(err)
	}
	hooks, ok := root["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks is not an object: %s", b)
	}
	groups, ok := hooks["UserPromptSubmit"].([]any)
	if !ok {
		t.Fatalf("UserPromptSubmit is not an array: %s", b)
	}
	var commands []string
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
			if !ok {
				continue
			}
			if command, ok := handler["command"].(string); ok {
				commands = append(commands, command)
			}
		}
	}
	return commands, groups
}

func countValue(values []string, target string) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}
	return count
}

func assertMatcherGroupPreserved(t *testing.T, groups []any) {
	t.Helper()
	for _, raw := range groups {
		group, ok := raw.(map[string]any)
		if !ok || group["matcher"] != "preserve-me" {
			continue
		}
		future, ok := group["future"].(map[string]any)
		if !ok || future["enabled"] != true {
			t.Fatalf("matcher group unknown fields changed: %#v", group)
		}
		handlers, ok := group["hooks"].([]any)
		if !ok || len(handlers) != 0 {
			t.Fatalf("matcher group hooks = %#v, want empty array", group["hooks"])
		}
		return
	}
	t.Fatal("matcher group was removed")
}

func TestDoctorReportsInvalidConfig(t *testing.T) {
	t.Setenv("PANESTRA_CLI_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Dir(config.Path()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.Path(), []byte("enabled = false\nauto_tmux = nope\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	if doctor(&output) {
		t.Fatal("doctor() = true for invalid config")
	}
	for _, want := range []string{"configuration could not be loaded", config.Path(), "toml:"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("doctor output does not contain %q:\n%s", want, output.String())
		}
	}
}

func TestWriteTerminalHints(t *testing.T) {
	tests := []struct {
		name        string
		termProgram string
		lcTerminal  string
		wantHint    bool
	}{
		{name: "direct iTerm2 session", termProgram: "iTerm.app", wantHint: true},
		{name: "iTerm2 session inside tmux", termProgram: "tmux", lcTerminal: "iTerm2", wantHint: true},
		{name: "other terminal", termProgram: "Apple_Terminal", wantHint: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TERM_PROGRAM", tt.termProgram)
			t.Setenv("LC_TERMINAL", tt.lcTerminal)
			var output strings.Builder
			writeTerminalHints(&output)
			got := output.String()
			if tt.wantHint {
				for _, want := range []string{"iTerm2 scrolling requires", "Enable mouse reporting", "Report mouse wheel events"} {
					if !strings.Contains(got, want) {
						t.Fatalf("terminal hint does not contain %q: %q", want, got)
					}
				}
			} else if got != "" {
				t.Fatalf("terminal hint = %q, want no output", got)
			}
		})
	}
}

func TestSetupRemovesStaleAgentConfiguration(t *testing.T) {
	home, fakeBin := t.TempDir(), t.TempDir()
	t.Setenv("PANESTRA_CLI_HOME", home)
	t.Setenv("PANESTRA_CLI_ALLOW_UNSUPPORTED", "1")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("SHELL", "/bin/zsh")
	for _, agent := range []string{"codex", "claude"} {
		if err := os.WriteFile(filepath.Join(fakeBin, agent), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", fakeBin)
	if err := Setup(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(fakeBin, "claude")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", config.ShimDir()+string(os.PathListSeparator)+fakeBin)
	if err := Setup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(config.ShimDir(), "claude")); !os.IsNotExist(err) {
		t.Fatal("stale Claude shim remains")
	}
	if adapterInstalled(claudeSettingsPath()) {
		t.Fatal("stale Claude adapter remains")
	}
	if !adapterInstalled(codexHooksPath()) {
		t.Fatal("Codex adapter was removed")
	}
}
