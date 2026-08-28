package tools

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
)

// typstImagePath returns a path rooted at the Typst project root configured by
// CompileTypst's --root argument.
func typstImagePath(imageName string) (string, error) {
	if err := safePathComponent(imageName); err != nil {
		return "", errors.New("invalid Typst image filename")
	}
	imageDir := filepath.Clean(config.ImageSavePath)
	if filepath.IsAbs(imageDir) || imageDir == "." || imageDir == ".." || strings.HasPrefix(imageDir, ".."+string(filepath.Separator)) {
		return "", errors.New("image storage must be inside the Typst project root")
	}
	return "/" + filepath.ToSlash(filepath.Join(imageDir, imageName)), nil
}
