package tools

import (
	"context"
	"log"
	"os/exec"
)

func MergePdf(inputs []string, outputName string) error {
	args := append(append([]string(nil), inputs...), outputName)

	ctx, cancel := context.WithTimeout(context.Background(), externalCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pdfunite", args...)
	if err := cmd.Run(); err != nil {
		log.Printf("From MergePdf can't merge pdf : %v", err)
		return err
	}

	return nil
}
