package installer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fru180/Panestra-cli/internal/config"
)

const beginMarker = "# >>> panestra-cli >>>"
const endMarker = "# <<< panestra-cli <<<"

func zshrcPath() string { return filepath.Join(config.Home(), ".zshrc") }

func addPathBlock() error {
	path := zshrcPath()
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := removeManagedBlock(string(b))
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += beginMarker + "\nexport PATH=\"$HOME/.local/share/panestra-cli/shims:$PATH\"\n" + endMarker + "\n"
	return atomicWrite(path, []byte(content), 0600)
}

func removePathBlock() error {
	path := zshrcPath()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return atomicWrite(path, []byte(removeManagedBlock(string(b))), 0600)
}

func removeManagedBlock(s string) string {
	for {
		start := strings.Index(s, beginMarker)
		if start < 0 {
			break
		}
		endRel := strings.Index(s[start:], endMarker)
		if endRel < 0 {
			break
		}
		end := start + endRel + len(endMarker)
		if end < len(s) && s[end] == '\n' {
			end++
		}
		s = s[:start] + s[end:]
	}
	return s
}
