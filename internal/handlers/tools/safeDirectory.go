package tools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ensureDirectoryTree verifies each path component with Lstat so directory
// operations do not traverse symlinked parents. Missing components are created
// one at a time when create is true.
func ensureDirectoryTree(path string, create bool, mode os.FileMode) error {
	clean := filepath.Clean(path)
	volume := filepath.VolumeName(clean)
	remainder := strings.TrimPrefix(clean, volume)
	current := volume
	if filepath.IsAbs(clean) {
		current += string(filepath.Separator)
		remainder = strings.TrimLeft(remainder, `/\`)
	}

	for _, component := range strings.FieldsFunc(remainder, func(r rune) bool { return r == '/' || r == '\\' }) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) && create {
			if err := os.Mkdir(current, mode); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
			info, err = os.Lstat(current)
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("unsafe directory component %q", current)
		}
	}
	return nil
}

func validateRegularFile(root, filename string) (string, os.FileInfo, error) {
	if err := safePathComponent(filename); err != nil {
		return "", nil, err
	}
	if err := ensureDirectoryTree(root, false, 0); err != nil {
		return "", nil, err
	}
	path := filepath.Join(root, filename)
	info, err := os.Lstat(path)
	if err != nil {
		return "", nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", nil, errors.New("unsafe non-regular file")
	}
	return path, info, nil
}
