package tools

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
)

// Vérification sécurité (taille, séparateur, etc.)
func CheckPdfFile(r *http.Request, maxBytes int64) (multipart.File, error) {
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
		file.Close()
		return nil, errors.New("not a valid PDF")
	}

	// Rewind pour que le reste du code puisse relire depuis le début
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			file.Close()
			return nil, errors.New("cannot rewind PDF")
		}
	} else {
		file.Close()
		return nil, errors.New("file is not seekable")
	}

	return file, nil
}
