package tools

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/config"
)

var markedExamPDFName = regexp.MustCompile(`^student-exam-([0-9]+)\.pdf$`)

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
	markExamsByID := make(map[int64]config.MarkExam, len(markExams))
	for _, exam := range markExams {
		if exam.StudentExamID <= 0 {
			log.Printf("Invalid student exam ID in marking result: %d", exam.StudentExamID)
			return "", false
		}
		if _, exists := markExamsByID[exam.StudentExamID]; exists {
			log.Printf("Duplicate student exam ID in marking results: %d", exam.StudentExamID)
			return "", false
		}
		markExamsByID[exam.StudentExamID] = exam
	}

	for _, file := range pdfFiles {
		base := filepath.Base(file)
		parts := markedExamPDFName.FindStringSubmatch(base)
		if parts == nil {
			log.Printf("Unexpected marked PDF name: %s", base)
			return "", false
		}

		studentExamID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			log.Printf("Invalid student exam ID in marked PDF name %s: %v", base, err)
			return "", false
		}
		if studentExamID <= 0 {
			log.Printf("Invalid student exam ID in marked PDF name %s: must be positive", base)
			return "", false
		}
		exam, exists := markExamsByID[studentExamID]
		if !exists {
			log.Printf("No marking result for student exam ID %d", studentExamID)
			return "", false
		}

		studentName := exam.FirstName + " " + exam.LastName
		add := fmt.Sprintf("%s, \"%d\",", typstStringLiteral(studentName), pageNumber)
		content += add
		pageNumber += exam.Pages
	}

	content += ")"

	_, err = output.WriteString(content)
	if err != nil {
		log.Printf("Can't write content, error : %v", err)
		return "", false
	}

	return typstFilePath, true
}
