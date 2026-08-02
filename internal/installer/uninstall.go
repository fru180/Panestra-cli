package installer

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/fru180/Panestra-cli/internal/config"
)

func Uninstall() error {
	var errs []error
	if err := uninstallAdapters(); err != nil {
		errs = append(errs, err)
	}
	if err := removePathBlock(); err != nil {
		errs = append(errs, err)
	}
	if err := os.RemoveAll(config.DataDir()); err != nil {
		errs = append(errs, err)
	}
	if err := os.RemoveAll(filepath.Dir(config.Path())); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
