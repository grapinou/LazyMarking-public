package tools

import (
	"os"
	"path/filepath"
)

func ClearDir(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// Construit le chemin complet
		fullPath := filepath.Join(path, entry.Name())

		// Supprime fichier ou dossier (récursif)
		err := os.RemoveAll(fullPath)
		if err != nil {
			return err
		}
	}

	return nil
}
