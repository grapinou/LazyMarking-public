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

	dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dstFile, file); err != nil {
		_ = dstFile.Close()
		_ = os.Remove(dstPath)
		return err
	}
	if err := dstFile.Close(); err != nil {
		_ = os.Remove(dstPath)
		return err
	}
	return nil
}
