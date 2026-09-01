package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MarkingArtifactExists checks an expected artifact without creating or
// modifying the marking workspace.
func MarkingArtifactExists(username string, markingJobID int64, filename string) (bool, error) {
	if markingJobID <= 0 || safePathComponent(filename) != nil || !strings.EqualFold(filepath.Ext(filename), ".pdf") {
		return false, os.ErrNotExist
	}
	workspace, err := operationTempDir(username, "marking-"+strconv.FormatInt(markingJobID, 10))
	if err != nil {
		return false, err
	}
	if err := ensureDirectoryTree(workspace, false, 0); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	info, err := os.Lstat(filepath.Join(workspace, filename))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.Mode()&os.ModeSymlink == 0 && info.Mode().IsRegular(), nil
}
