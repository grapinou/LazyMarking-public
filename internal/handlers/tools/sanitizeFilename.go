package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

func SanitizeFilename(userID int64, username string, questionID int64, originalFilename string) (string, error) {
	// Garder l'extension (.jpg, .png, etc.)
	ext := strings.ToLower(filepath.Ext(originalFilename))
	if ext != ".jpg" && ext != ".png" && ext != ".svg" {
		return "", fmt.Errorf("extension non autorisée : %s", ext)
	}

	cleanFilename := filepath.Base(originalFilename)

	// Construire le nom du fichier
	filename := fmt.Sprintf("%d_%s_%d_%s", userID, username, questionID, cleanFilename)
	return filename, nil
}
