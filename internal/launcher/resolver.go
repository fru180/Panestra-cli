package launcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fru180/Panestra-cli/internal/config"
)

func Resolve(agent string) (string, error) {
	if agent != "codex" && agent != "claude" {
		return "", fmt.Errorf("unsupported agent %q", agent)
	}
	shim, _ := filepath.EvalSymlinks(filepath.Join(config.ShimDir(), agent))
	self, _ := os.Executable()
	self, _ = filepath.EvalSymlinks(self)
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, agent)
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Mode()&0111 == 0 {
			continue
		}
		real, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			real = candidate
		}
		if samePath(real, shim) || samePath(real, self) || samePath(dir, config.ShimDir()) {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("real %s executable was not found outside %s", agent, config.ShimDir())
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	aa, _ := filepath.Abs(a)
	bb, _ := filepath.Abs(b)
	return strings.TrimRight(aa, string(filepath.Separator)) == strings.TrimRight(bb, string(filepath.Separator))
}
