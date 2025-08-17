package tools

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
)

func ServePdf(username string, qcmType config.QCMType, w http.ResponseWriter) {
	typstName := username + string(qcmType)
	pdfName := strings.TrimSuffix(typstName, filepath.Ext(typstName)) + ".pdf"
	pdfPath := filepath.Join("assets", "tmp", username, pdfName)

	// Open file
	f, err := os.Open(pdfPath)
	if err != nil {
		log.Printf("From ServePdf -> Open, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Set header
	w.Header().Set("Content-type", "application/pdf")

	// Stream to response
	if _, err := io.Copy(w, f); err != nil {
		log.Printf("From ServePdf -> Copy, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
	}

}
