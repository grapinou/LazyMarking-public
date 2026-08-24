package tools

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func SplitPdf(file io.Reader, tmpDir, outputPattern string) error {
	// fichier temporaire dans ton dossier tmpDir
	tmpFile, err := os.CreateTemp(tmpDir, "input-*.pdf")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	// copie du Reader vers le fichier temporaire
	if _, err := io.Copy(tmpFile, file); err != nil {
		return err
	}
	tmpFile.Close()

	// exécution de pdfseparate
	ctx, cancel := context.WithTimeout(context.Background(), externalCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pdfseparate", tmpFile.Name(), filepath.Join(tmpDir, outputPattern))
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("pdfseparate failed: %v\nOutput: %s", err, out)
		return err
	}

	return nil
}
