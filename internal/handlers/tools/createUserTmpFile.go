package tools

import (
	"errors"
	"log"
	"os"
	"path/filepath"
)

// CreateOperationTempDir isolates files belonging to one generation or marking job.
func CreateOperationTempDir(username, operation string) (string, bool) {
	tempPath, err := operationTempDir(username, operation)
	if err != nil {
		return "", false
	}
	if err := os.MkdirAll(tempPath, 0o750); err != nil {
		log.Printf("From CreateOperationTempDir: %v", err)
		return "", false
	}
	return tempPath, true
}

func operationTempDir(username, operation string) (string, error) {
	if safePathComponent(username) != nil || safePathComponent(operation) != nil {
		return "", errors.New("invalid workspace component")
	}
	return filepath.Join("assets", "tmp", username, operation), nil
}

// RemoveOperationTempDir removes only the validated operation workspace. An
// already absent workspace is considered successfully removed.
func RemoveOperationTempDir(username, operation string) error {
	tempPath, err := operationTempDir(username, operation)
	if err != nil {
		return err
	}
	return os.RemoveAll(tempPath)
}
