package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type fileChange struct {
	path   string
	data   []byte
	mode   os.FileMode
	remove bool
}

type fileState struct {
	exists bool
	data   []byte
	mode   os.FileMode
}

func applyFileChanges(changes []fileChange) error {
	return applyFileChangesWith(changes, applyFileChange)
}

func applyFileChangesWith(changes []fileChange, apply func(fileChange) error) error {
	states := make(map[string]fileState, len(changes))
	seen := make(map[string]struct{}, len(changes))
	for _, change := range changes {
		if _, ok := seen[change.path]; ok {
			return fmt.Errorf("duplicate setup target %s", change.path)
		}
		seen[change.path] = struct{}{}
		state, err := readFileState(change.path)
		if err != nil {
			return err
		}
		states[change.path] = state
	}

	createdDirs, err := missingParentDirs(changes)
	if err != nil {
		return err
	}
	for i, change := range changes {
		if err := apply(change); err != nil {
			err = fmt.Errorf("apply setup change %s: %w", change.path, err)
			rollbackErr := rollbackFileChanges(changes[:i], states, createdDirs)
			if rollbackErr != nil {
				return errors.Join(err, fmt.Errorf("rollback setup changes: %w", rollbackErr))
			}
			return err
		}
	}
	return nil
}

func applyFileChange(change fileChange) error {
	if change.remove {
		if err := os.Remove(change.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return atomicWrite(change.path, change.data, change.mode)
}

func readFileState(path string) (fileState, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return fileState{}, nil
	}
	if err != nil {
		return fileState{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return fileState{}, err
	}
	return fileState{exists: true, data: data, mode: info.Mode().Perm()}, nil
}

func missingParentDirs(changes []fileChange) ([]string, error) {
	dirs := map[string]struct{}{}
	for _, change := range changes {
		for dir := filepath.Dir(change.path); ; dir = filepath.Dir(dir) {
			info, err := os.Stat(dir)
			if err == nil {
				if !info.IsDir() {
					return nil, fmt.Errorf("setup parent path is not a directory: %s", dir)
				}
				break
			}
			if !os.IsNotExist(err) {
				return nil, err
			}
			dirs[dir] = struct{}{}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	result := make([]string, 0, len(dirs))
	for dir := range dirs {
		result = append(result, dir)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.Count(result[i], string(filepath.Separator)) > strings.Count(result[j], string(filepath.Separator))
	})
	return result, nil
}

func rollbackFileChanges(applied []fileChange, states map[string]fileState, createdDirs []string) error {
	var errs []error
	for i := len(applied) - 1; i >= 0; i-- {
		change := applied[i]
		state := states[change.path]
		if state.exists {
			if err := atomicWrite(change.path, state.data, state.mode); err != nil {
				errs = append(errs, fmt.Errorf("restore %s: %w", change.path, err))
			}
		} else if err := os.Remove(change.path); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("remove %s: %w", change.path, err))
		}
	}
	for _, dir := range createdDirs {
		if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("remove directory %s: %w", dir, err))
		}
	}
	return errors.Join(errs...)
}
