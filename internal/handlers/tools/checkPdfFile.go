package tools

import (
	"errors"
	"io"
	"net/http"
)

// Vérification sécurité (taille, séparateur, etc.)
func CheckPdfFile(r *http.Request, maxBytes int64) (io.Reader, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		return nil, errors.New("file too large")
	}

	file, _, err := r.FormFile("pdffile")
	if err != nil {
		return nil, err
	}

	// Vérification magic bytes : on s'assure que c'est bien un pdf
	buf := make([]byte, 5)
	n, err := file.Read(buf)
	if err != nil || n < 5 || string(buf) != "%PDF-" {
		return nil, errors.New("not a valid PDF")
	}

	// Rewind pour que le reste du code puisse relire depuis le début
	if seeker, ok := file.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	} else {
		return nil, errors.New("file is not seekable")
	}

	return file, nil
}
