package tools

import (
	"log"
	"os"
)

func RemoveFiles(files []string) error {
	for _, file := range files {

		err := os.Remove(file) // nom du fichier à supprimer
		if err != nil {
			log.Printf("From RemoveFiles Error : %v", err)
			return err
		}
	}
	return nil
}
