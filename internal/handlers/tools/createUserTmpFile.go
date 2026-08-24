package tools

import (
	"errors"
	"log"
	"os"
	"path/filepath"
)

// CreateUserTempDir creates the user's root workspace.
func CreateUserTempDir(username string) (string, bool) {
	if err := safePathComponent(username); err != nil {
		log.Printf("From CreateUserTempDir: invalid username path component")
		return "", false
	}
	tempPath := filepath.Join("assets", "tmp", username)

	// MkdirAll ne renvoie pas d'erreur si le dossier existe déjà
	if err := os.MkdirAll(tempPath, 0o755); err != nil {
		log.Printf("From CreateUserTempDir : can't create user dir : error : %v", err)
		return "", false
	}

	return tempPath, true
}

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
