package tools

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
)

// RemoveStoredImageFile removes one validated image entry from the configured
// image directory. An already absent file is considered successfully removed.
func RemoveStoredImageFile(filename string) error {
	if safePathComponent(filename) != nil {
		return errors.New("invalid stored image filename")
	}

	err := os.Remove(filepath.Join(config.ImageSavePath, filename))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
