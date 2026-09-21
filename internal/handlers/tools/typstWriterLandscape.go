package tools

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TypstWriterLandscape(tempDir, username string, qcm config.QCM) (string, bool) {
	refQCMTypst := config.RefQCMLandscapeTypst // fichier existant

	// 1. Ouvrir l'ancien fichier pour lecture
	input, err := os.Open(refQCMTypst)
	if err != nil {
		log.Printf("Can't open file from TypstWriterLandscape : %s, error : %v", refQCMTypst, err)
		return "", false
	}
	defer input.Close()

	// 2. Créer le nouveau fichier (écrasement s’il existe)
	typstFilePath := filepath.Join(tempDir, fmt.Sprintf("%s%v", username, config.PreviewLandscapeQCM))
	output, err := os.Create(typstFilePath)
	if err != nil {
		log.Printf("Can't create file from TypstWriterLandscape: %s, error : %v", typstFilePath, err)
		return "", false
	}
	defer output.Close()

	// 4. Copier le contenu de l’ancien fichier
	_, err = io.Copy(output, input)
	if err != nil {
		log.Printf("Can't copy ref file into from TypstWriterLandscape: %s, error : %v", typstFilePath, err)
		return "", false
	}

	content, err := TypstLandscapeContent(qcm)
	if err != nil {
		log.Printf("Can't build landscape content: %v", err)
		return "", false
	}
	if _, err = output.WriteString(content); err != nil {
		return "", false
	}

	return typstFilePath, true
}
