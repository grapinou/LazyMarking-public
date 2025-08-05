package tools

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
)

func SanitizeFilename(userID int64, username string, questionType config.QuestionType, questionID int64, originalFilename string) (string, error) {
	if questionType != config.MainQuestion && questionType != config.AltQuestion {
		return "", fmt.Errorf("questionType invalid : %s", questionType)
	}

	// Garder l'extension (.jpg, .png, etc.)
	ext := strings.ToLower(filepath.Ext(originalFilename))
	if ext != ".jpg" && ext != ".png" && ext != ".svg" {
		return "", fmt.Errorf("ext not allowed : %s", ext)
	}

	cleanFilename := filepath.Base(originalFilename)

	// Construire le nom du fichier
	filename := fmt.Sprintf("%d_%s_%s_%d_%s", userID, username, questionType, questionID, cleanFilename)
	return filename, nil
}
