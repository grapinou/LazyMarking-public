package tools

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TypstBuildContent(tempDir string, markExams []config.MarkExam, pdfFiles []string) (string, bool) {
	refContentTypst := config.RefContentTypst // fichier existant

	// 1. Ouvrir l'ancien fichier pour lecture
	input, err := os.Open(refContentTypst)
	if err != nil {
		log.Printf("Can't open file : %s, error : %v", refContentTypst, err)
		return "", false
	}
	defer input.Close()

	// 2. Créer le nouveau fichier (écrasement s’il existe)
	typstFilePath := filepath.Join(tempDir, "00_content.typ")
	output, err := os.Create(typstFilePath)
	if err != nil {
		log.Printf("Can't create file : %s, error : %v", typstFilePath, err)
		return "", false
	}
	defer output.Close()

	// 4. Copier le contenu de l’ancien fichier
	_, err = io.Copy(output, input)
	if err != nil {
		log.Printf("Can't copy ref file into : %s, error : %v", typstFilePath, err)
		return "", false
	}

	// 5. Ajouter des lignes à la fin
	content := "\n"
	pageNumber := 1
	for _, file := range pdfFiles {
		// Récupère le nom de fichier (Alice_Dupont.pdf)
		base := filepath.Base(file)
		// Enlève l'extension (.pdf)
		name := strings.TrimSuffix(base, filepath.Ext(base)) // Alice_Dupont
		// Sépare prénom et nom
		parts := strings.SplitN(name, "_", 2) // ["Alice", "Dupont"]
		if len(parts) != 2 {
			log.Printf("Unexpected marked PDF name: %s", base)
			return "", false
		}

		firstName := parts[0]
		lastName := parts[1]

		for _, exam := range markExams {
			if exam.FirstName == firstName && exam.LastName == lastName {
				add := fmt.Sprintf("\"%s %s\", \"%d\",", exam.FirstName, exam.LastName, pageNumber)
				content += add
				pageNumber += exam.Pages
			}
		}

	}

	content += ")"

	_, err = output.WriteString(content)
	if err != nil {
		log.Printf("Can't write content, error : %v", err)
		return "", false
	}

	return typstFilePath, true
}
