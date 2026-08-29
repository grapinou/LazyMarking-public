package tools

import (
	"errors"

	"github.com/grapinou/LazyMarking/internal/filesafety"
)

func safePathComponent(value string) error {
	if !filesafety.IsSafePathComponent(value) {
		return errors.New("invalid path component")
	}
	return nil
}
