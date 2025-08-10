package tools

import (
	"errors"
	"io"
	"net/http"
)

// Vérification sécurité (taille, séparateur, etc.)
func CheckCSVFile(r *http.Request, maxBytes int64) (io.Reader, error) {
	// Limite stricte de taille
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)

	// Parse multipart
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		return nil, errors.New("file too large")
	}

	// Récupération du fichier
	file, _, err := r.FormFile("csvfile")
	if err != nil {
		return nil, err
	}

	return file, nil
}
