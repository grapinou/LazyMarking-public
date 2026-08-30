package tools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var ErrAmbiguousExamGenerationPDF = errors.New("multiple exam generation PDFs")

// ResolveExamGenerationPDFName returns the sole regular PDF stored in the
// generation workspace. Successful generation cleanup leaves only this final
// artifact; refusing ambiguity avoids selecting an arbitrary historical file.
func ResolveExamGenerationPDFName(username string, generationID int64) (string, error) {
	if generationID <= 0 {
		return "", os.ErrNotExist
	}
	operation := "exam-" + strconv.FormatInt(generationID, 10)
	workspace, err := operationTempDir(username, operation)
	if err != nil {
		return "", err
	}
	if err := ensureDirectoryTree(workspace, false, 0); err != nil {
		return "", err
	}

	entries, err := os.ReadDir(workspace)
	if err != nil {
		return "", err
	}
	var pdfName string
	for _, entry := range entries {
		name := entry.Name()
		if entry.Type()&os.ModeSymlink != 0 || !strings.EqualFold(filepath.Ext(name), ".pdf") || safePathComponent(name) != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if pdfName != "" {
			return "", fmt.Errorf("%w in %s", ErrAmbiguousExamGenerationPDF, operation)
		}
		pdfName = name
	}
	if pdfName == "" {
		return "", os.ErrNotExist
	}
	return pdfName, nil
}
