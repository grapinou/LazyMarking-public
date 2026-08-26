package generateexams

import (
	"log"
	"strings"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

func examGenerationPDFName(username, examName, classCodeName string) string {
	return safeExamFilenamePart(username) + "_exam_" + safeExamFilenamePart(examName) + "_" + safeExamFilenamePart(classCodeName) + ".pdf"
}

func safeExamFilenamePart(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range value {
		if r == '/' || r == '\\' || r < 0x20 || r == 0x7f {
			builder.WriteByte('_')
			continue
		}
		builder.WriteRune(r)
	}

	result := builder.String()
	if result == "" || result == "." || result == ".." {
		return "_"
	}
	return result
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
