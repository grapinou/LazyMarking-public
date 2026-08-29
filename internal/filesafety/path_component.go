package filesafety

import (
	"path/filepath"
	"strings"
)

func IsSafePathComponent(value string) bool {
	return value != "" && value != "." && value != ".." &&
		filepath.Base(value) == value && !strings.ContainsAny(value, "/\\\x00")
}
