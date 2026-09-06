package tools

import (
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func ServePdfNamed(username, operation, filename string, w http.ResponseWriter, r *http.Request) {
	servePdfNamed(username, operation, filename, filename, "inline", w, r)
}

func ServePdfDownloadNamed(username, operation, storageFilename, downloadFilename string, w http.ResponseWriter, r *http.Request) {
	servePdfNamed(username, operation, storageFilename, downloadFilename, "attachment", w, r)
}

func servePdfNamed(username, operation, filename, displayFilename, disposition string, w http.ResponseWriter, r *http.Request) {
	if safePathComponent(username) != nil || safePathComponent(operation) != nil || safePathComponent(filename) != nil || !strings.EqualFold(filepath.Ext(filename), ".pdf") {
		http.Error(w, "Le nom du PDF est invalide.", http.StatusBadRequest)
		return
	}

	workspace, err := operationTempDir(username, operation)
	if err != nil {
		http.Error(w, "Le nom du PDF est invalide.", http.StatusBadRequest)
		return
	}
	if err := ensureDirectoryTree(workspace, false, 0); err != nil {
		log.Printf("From ServePdf -> unsafe workspace: %v", err)
		http.Error(w, "Le PDF est introuvable.", http.StatusNotFound)
		return
	}
	pdfPath := filepath.Join(workspace, filename)

	lstatInfo, err := os.Lstat(pdfPath)
	if err != nil {
		log.Printf("From ServePdf -> Lstat failed: %v", err)
		http.Error(w, "Le PDF est introuvable.", http.StatusNotFound)
		return
	}
	if lstatInfo.Mode()&os.ModeSymlink != 0 || !lstatInfo.Mode().IsRegular() {
		log.Printf("From ServePdf -> refusing non-regular PDF path: %s", pdfPath)
		http.Error(w, "Le PDF est introuvable.", http.StatusNotFound)
		return
	}

	// Open file
	f, err := os.Open(pdfPath)
	if err != nil {
		log.Printf("From ServePdf -> Open, error : %v", err)
		http.Error(w, "Le PDF est introuvable.", http.StatusNotFound)
		return
	}
	defer f.Close()
	openedInfo, err := f.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || !os.SameFile(lstatInfo, openedInfo) {
		log.Printf("From ServePdf -> opened file does not match validated PDF path: %s", pdfPath)
		http.Error(w, "Le PDF est introuvable.", http.StatusNotFound)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", "application/pdf")
	// Inline pour affichage dans le navigateur, attachment pour forcer téléchargement
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": displayFilename}))
	http.ServeContent(w, r, displayFilename, openedInfo.ModTime(), f)
}
