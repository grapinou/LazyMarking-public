package tools

import (
	"errors"
	"path/filepath"
	"strings"
)

func safePathComponent(value string) error {
	if value == "" || value == "." || value == ".." || filepath.Base(value) != value {
		return errors.New("invalid path component")
	}
	if strings.ContainsAny(value, `/\x00`) {
		return errors.New("invalid path component")
	}
	return nil
}
