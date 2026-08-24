package tools

import (
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

func SaveUploadedFile(file multipart.File, dstFolder, filename string) error {
	if safePathComponent(filename) != nil {
		return errors.New("invalid upload filename")
	}
	// S’assurer que le dossier existe
	err := os.MkdirAll(dstFolder, 0o750)
	if err != nil {
		return err
	}

	dstPath := filepath.Join(dstFolder, filename)

	dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, file)
	return err
}
