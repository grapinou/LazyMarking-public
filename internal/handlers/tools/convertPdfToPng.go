package tools

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

func ConvertPdfToPng(tempDir, pdfName, outputPrefix string) string {
	pdfPath := filepath.Join(tempDir, pdfName)
	pdfFile := pdfPath

	name := strings.TrimSuffix(pdfName, filepath.Ext(pdfName))
	outputName := outputPrefix + name
	outputPath := filepath.Join(tempDir, outputName)

	// Commande équivalente à : pdftoppm -png input.pdf page
	cmd := exec.Command("pdftoppm", "-png", "-singlefile", pdfFile, outputPath)

	// Exécuter la commande
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Erreur conversion PDF -> PNG: %v", err)
	}

	fmt.Println("Conversion terminée !")

	return outputName + ".png"
}
