package tools

import (
	"log"
	"os/exec"
)

func MergePdf(inputs []string, outputName string) error {
	args := append(inputs, outputName)

	cmd := exec.Command("pdfunite", args...)
	if err := cmd.Run(); err != nil {
		log.Printf("From MergePdf can't merge pdf : %v", err)
		return err
	}

	return nil
}
