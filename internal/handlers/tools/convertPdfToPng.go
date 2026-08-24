package tools

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func ConvertPdfToPng(tempDir, pdfName, outputPrefix string) (string, error) {
	pdfPath := filepath.Join(tempDir, pdfName)
	pdfFile := pdfPath

	name := strings.TrimSuffix(pdfName, filepath.Ext(pdfName))
	outputName := outputPrefix + name
	outputPath := filepath.Join(tempDir, outputName)

	// Commande équivalente à : pdftoppm -png input.pdf page
	ctx, cancel := context.WithTimeout(context.Background(), externalCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pdftoppm", "-png", "-singlefile", pdfFile, outputPath)

	// Exécuter la commande
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("convert PDF to PNG: %w: %s", err, output)
	}

	return outputName + ".png", nil
}
