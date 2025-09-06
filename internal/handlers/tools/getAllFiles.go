package tools

import (
	"log"
	"path/filepath"
)

func GetAllFiles(dir, ext string) ([]string, error) {
	folder := dir  // le dossier à explorer
	pattern := ext // "*.pdf"  le type de fichiers recherché

	// filepath.Glob renvoie tous les fichiers correspondant au pattern
	files, err := filepath.Glob(filepath.Join(folder, pattern))
	if err != nil {
		log.Printf("can't get all files, error : %v", err)
		return files, err
	}

	return files, nil
}
