package tools

import (
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func ServePdfNamed(username, operation, filename string, w http.ResponseWriter) {
	if safePathComponent(username) != nil || safePathComponent(operation) != nil || safePathComponent(filename) != nil || !strings.EqualFold(filepath.Ext(filename), ".pdf") {
		http.Error(w, "Invalid PDF name", http.StatusBadRequest)
		return
	}

	pdfPath := filepath.Join("assets", "tmp", username, operation, filename)

	// Open file
	f, err := os.Open(pdfPath)
	if err != nil {
		log.Printf("From ServePdf -> Open, error : %v", err)
		http.Error(w, "PDF not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.Error(w, "PDF not found", http.StatusNotFound)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", "application/pdf")
	// Inline pour affichage dans le navigateur, attachment pour forcer téléchargement
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": filename}))
	http.ServeContent(w, nil, filename, info.ModTime(), f)
}
