package tools

import (
	"fmt"
	"mime/multipart"
	"net/http"
)

const MaxCSVRequestBytes int64 = 2 << 20

// Vérification sécurité (taille, séparateur, etc.)
func CheckCSVFile(w http.ResponseWriter, r *http.Request, maxBytes int64) (multipart.File, error) {
	// The limit applies to the complete multipart request, including its envelope.
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		removeCSVMultipartFiles(r)
		return nil, fmt.Errorf("parse CSV multipart form: %w", err)
	}

	file, _, err := r.FormFile("csvfile")
	if err != nil {
		removeCSVMultipartFiles(r)
		return nil, err
	}

	return file, nil
}

func removeCSVMultipartFiles(r *http.Request) {
	if r.MultipartForm != nil {
		_ = r.MultipartForm.RemoveAll()
	}
}
