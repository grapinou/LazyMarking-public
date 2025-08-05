package tools

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

// DeleteImageFile supprime un fichier image donné par l'id de la question
// dans une utilisation qui dépasse celle des images d'où la vérification
// qu'une question contient ou non une image
func DeleteImageFile(userID, questionID int64, w http.ResponseWriter, r *http.Request, queries *db.Queries) error {
	image, err := queries.GetImageByQuestionID(r.Context(), db.GetImageByQuestionIDParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		log.Printf("From DeleteImageFile -> GetImageByQuestionID DB error: %v", err)
		return err
	}

	path := filepath.Join(config.ImageSavePath, image.ImageName)
	err = os.Remove(path)
	if err != nil {
		log.Printf("tools.DeleteImageFile : can't delete file %s : %v", path, err)
		return err
	}
	return nil
}
