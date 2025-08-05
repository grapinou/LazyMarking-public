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

// DeleteAltImageFile supprime un fichier alt image donné par l'id de la alt question
// dans une utilisation qui dépasse celle des images d'où la vérification
// qu'une question contient ou non une image
func DeleteAltImageFile(userID, altQuestionID int64, w http.ResponseWriter, r *http.Request, queries *db.Queries) error {
	image, err := queries.GetAltImageByAltQuestionID(r.Context(), db.GetAltImageByAltQuestionIDParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
	})
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		log.Printf("From DeleteAltImageFile -> GetAltImageByAltQuestionID DB error: %v", err)
		return err
	}

	path := filepath.Join(config.ImageSavePath, image.ImageName)
	err = os.Remove(path)
	if err != nil {
		log.Printf("tools.DeleteAltImageFile : can't delete file %s : %v", path, err)
		return err
	}
	return nil
}
