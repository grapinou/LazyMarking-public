package tools

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// CompileTypst génère un PDF à partir d’un fichier Typst (.typ)
// et retourne le chemin vers le PDF si succès, sinon "".
func CompileTypst(typstPath string) (string, bool) {
	projectRoot, err := os.Getwd()
	if err != nil {
		log.Printf("Erreur Getwd : %v", err)
		return "", false
	}

	// Exemple : assets/tmp/alice/alice_preview.typ
	dir := filepath.Dir(typstPath)
	base := filepath.Base(typstPath)
	pdfPath := filepath.Join(dir, base[:len(base)-4]+".pdf") // change .typ en .pdf

	cmd := exec.Command("typst", "compile", "--root", projectRoot, typstPath, pdfPath)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error with compile typst. Can't make pdf, error : %v\n%s", err, out)
		return "", false
	}

	return pdfPath, true
}
