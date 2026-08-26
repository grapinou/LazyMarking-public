package generateexams

import (
	"log"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

func examGenerationPDFName(username, examName, classCodeName string) string {
	return username + "_exam_" + examName + "_" + classCodeName + ".pdf"
}

func cleanupExamGenerationFiles(tempDir string, pdfFiles []string) {
	if err := tools.RemoveFiles(pdfFiles); err != nil {
		log.Printf("From GenerateExamsHandler -> cleanup intermediate PDFs: %v", err)
	}

	for _, pattern := range []string{"*.png", "*.typ"} {
		files, err := tools.GetAllFiles(tempDir, pattern)
		if err != nil {
			log.Printf("From GenerateExamsHandler -> list cleanup files %s: %v", pattern, err)
			continue
		}
		if err := tools.RemoveFiles(files); err != nil {
			log.Printf("From GenerateExamsHandler -> cleanup files %s: %v", pattern, err)
		}
	}
}
