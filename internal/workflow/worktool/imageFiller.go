package worktool

import (
	"bytes"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func PostImageTesterWF(baseURL, urlTested string, image ImageStructWf) {
	log.Println("---------")
	log.Println("Testting : Image")

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Ajouter l'image
	if image.ImagePath != "" {
		file, err := os.Open(image.ImagePath)
		if err != nil {
			log.Fatalf("❌ Impossible d'ouvrir le fichier image %s: %v", image.ImagePath, err)
		}
		defer file.Close()

		// Créer un champ de formulaire pour le fichier
		part, err := writer.CreateFormFile("image", filepath.Base(image.ImagePath))
		if err != nil {
			log.Fatalf("❌ Erreur création champ fichier: %v", err)
		}
		if _, err := io.Copy(part, file); err != nil {
			log.Fatalf("❌ Erreur copie du fichier dans la requête: %v", err)
		}

		_ = writer.WriteField("question_id", image.QuestionID)
		_ = writer.WriteField("width", image.Width)
		if image.AltQuestionID != "" {
			_ = writer.WriteField("alt_question_id", image.AltQuestionID)
		}
	}
	// Fin de l'écriture du body multipart
	if err := writer.Close(); err != nil {
		log.Fatalf("❌ Erreur fermeture writer multipart: %v", err)
	}

	// Créer la requête POST
	postURL := baseURL + urlTested
	req, err := http.NewRequest("POST", postURL, &body)
	if err != nil {
		log.Fatalf("❌ Erreur création requête POST %s: %v", postURL, err)
	}

	// Définir le Content-Type à multipart/form-data avec la boundary générée automatiquement
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Envoyer la requête
	resp, err := Client.Do(req)
	if err != nil {
		log.Fatalf("❌ POST %s échoué: %v", postURL, err)
	}
	defer resp.Body.Close()

	// Vérification du statut
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("❌ POST %s échoué: statut %d", postURL, resp.StatusCode)
	}

	log.Printf("✅ POST %s : succès (statut %d, redirigé vers %s)\n",
		postURL, resp.StatusCode, resp.Request.URL.Path)
}
