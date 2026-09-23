package tools

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CompileTypst génère un PDF à partir d’un fichier Typst (.typ)
// et retourne le chemin vers le PDF si succès, sinon "".
func CompileTypst(typstPath string) (string, bool) {
	pdfPath, err := CompileTypstDetailed(typstPath)
	if err != nil {
		log.Printf("Error with compile typst: %v", err)
		return "", false
	}
	return pdfPath, true
}

// CompileTypstDetailed keeps the normal compilation path while exposing one
// short compiler diagnostic to the authenticated teacher preview.
func CompileTypstDetailed(typstPath string) (string, error) {
	projectRoot, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Exemple : assets/tmp/alice/alice_preview.typ
	dir := filepath.Dir(typstPath)
	base := filepath.Base(typstPath)
	pdfPath := filepath.Join(dir, base[:len(base)-4]+".pdf") // change .typ en .pdf

	ctx, cancel := context.WithTimeout(context.Background(), externalCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "typst", "compile", "--root", projectRoot, typstPath, pdfPath)

	out, err := cmd.CombinedOutput()
	if err != nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "error:") {
				if len(line) > 240 {
					line = line[:240]
				}
				return "", fmt.Errorf("%s", line)
			}
		}
		return "", fmt.Errorf("compilation Typst impossible : %w", err)
	}

	return pdfPath, nil
}
