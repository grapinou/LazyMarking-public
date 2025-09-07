package tools

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func ServePdfNamed(username, filename string, w http.ResponseWriter) {
	pdfPath := filepath.Join("assets", "tmp", username, filename)

	// Open file
	f, err := os.Open(pdfPath)
	if err != nil {
		log.Printf("From ServePdf -> Open, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Set headers
	w.Header().Set("Content-Type", "application/pdf")
	// Inline pour affichage dans le navigateur, attachment pour forcer téléchargement
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))

	// Stream to response
	if _, err := io.Copy(w, f); err != nil {
		log.Printf("From ServePdf -> Copy, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
	}
}
