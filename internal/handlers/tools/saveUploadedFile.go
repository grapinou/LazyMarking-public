package tools

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

func SaveUploadedFile(file multipart.File, dstFolder, filename string) error {
	// S’assurer que le dossier existe
	err := os.MkdirAll(dstFolder, os.ModePerm)
	if err != nil {
		return err
	}

	dstPath := filepath.Join(dstFolder, filename)

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, file)
	return err
}
