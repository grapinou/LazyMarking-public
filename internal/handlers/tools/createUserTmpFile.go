package tools

import (
	"log"
	"os"
	"path/filepath"
)

// CreateUserTempDir crée un dossier temporaire du type: assets/tmp/<username>/
func CreateUserTempDir(username string) (string, bool) {
	tempPath := filepath.Join("assets", "tmp", username)

	// MkdirAll ne renvoie pas d'erreur si le dossier existe déjà
	if err := os.MkdirAll(tempPath, 0o755); err != nil {
		log.Printf("From CreateUserTempDir : can't create user dir : error : %v", err)
		return "", false
	}

	return tempPath, true
}
