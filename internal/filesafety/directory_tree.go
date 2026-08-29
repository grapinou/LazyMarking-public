package filesafety

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ValidateDirectoryTree(path string) error {
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
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("unsafe directory component %q", current)
		}
	}
	return nil
}
