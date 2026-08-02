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
	created := filepath.Join(root, "new", "nested", "created")
	for _, file := range []struct {
		path    string
		content string
		mode    os.FileMode
	}{
		{path: existing, content: "before\n", mode: 0640},
		{path: stale, content: "restore me\n", mode: 0604},
	} {
		if err := os.WriteFile(file.path, []byte(file.content), file.mode); err != nil {
			t.Fatal(err)
		}
	}
	before := snapshotTree(t, root)
	wantErr := errors.New("injected commit failure")
	changes := []fileChange{
		{path: existing, data: []byte("after\n"), mode: 0600},
		{path: stale, remove: true},
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
