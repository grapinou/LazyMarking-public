package tools

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

func ExportTypstToPNGs(typstPath string) ([]string, bool) {
	projectRoot, err := os.Getwd()
	if err != nil {
		log.Printf("Erreur Getwd : %v", err)
		return nil, false
	}

	dir := filepath.Dir(typstPath)
	base := filepath.Base(typstPath)

	fmt.Println(base)

	// Sans extension .typ
	baseName := base[:len(base)-len(filepath.Ext(base))]
	fmt.Println(baseName)

	// Fichier de sortie principal
	// ⚠ Typst générera baseName.png ou baseName-1.png, baseName-2.png...
	pngBase := filepath.Join(dir, baseName+"page-{0p}-of-{t}.png")

	cmd := exec.Command("typst", "compile", "--root", projectRoot, "--format", "png", "--ppi", "300", typstPath, pngBase)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error with export typst. Can't make png, error : %v\n%s", err, out)
		return nil, false
	}

	// Récupérer toutes les images générées
	files, err := filepath.Glob(filepath.Join(dir, baseName+"*.png"))
	if err != nil {
		log.Printf("Erreur lors de la récupération des PNG générés : %v", err)
		return nil, false
	}

	sort.Strings(files) // pour garantir l’ordre des pages
	fmt.Println(files)

	return files, true
}
