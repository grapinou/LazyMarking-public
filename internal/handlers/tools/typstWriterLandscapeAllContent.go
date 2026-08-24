package tools

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TypstWriterLandscapeAllContent(tempDir, username string, allContent []string) (string, bool) {
	refQCMTypst := config.RefQCMLandscapeTypst // fichier existant

	// 1. Ouvrir l'ancien fichier pour lecture
	input, err := os.Open(refQCMTypst)
	if err != nil {
		log.Printf("Can't open file from TypstWriterLandscapeAllContent : %s, error : %v", refQCMTypst, err)
		return "", false
	}
	defer input.Close()

	// 2. Créer le nouveau fichier (écrasement s’il existe)
	typstFilePath := filepath.Join(tempDir, fmt.Sprintf("%s%v", username, config.MiniQCM))
	output, err := os.Create(typstFilePath)
	if err != nil {
		log.Printf("Can't create file from TypstWriterLandscapeAllContent: %s, error : %v", typstFilePath, err)
		return "", false
	}
	defer output.Close()

	// 4. Copier le contenu de l’ancien fichier
	_, err = io.Copy(output, input)
	if err != nil {
		log.Printf("Can't copy ref file into from TypstWriterLandscapeAllContent: %s, error : %v", typstFilePath, err)
		return "", false
	}

	// 5. Ajouter des lignes à la fin
	for _, content := range allContent {
		_, err = output.WriteString(content)
		if err != nil {
			log.Printf("Can't write content from TypstWriterLandscapeAllContent, error : %v", err)
			return "", false
		}
	}

	return typstFilePath, true
}
