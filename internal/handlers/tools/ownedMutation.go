package tools

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
)

func HandleOwnedMutationRows(w http.ResponseWriter, rows int64, operation string) bool {
	switch rows {
	case 1:
		return true
	case 0:
		http.Error(w, "La ressource demandée est introuvable.", http.StatusNotFound)
	default:
		log.Printf("%s integrity anomaly: affected %d rows", operation, rows)
		http.Error(w, "Impossible d’accéder aux données demandées.", http.StatusInternalServerError)
	}
	return false
}

func HandleOwnedLookupError(w http.ResponseWriter, err error, operation string) {
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "La ressource demandée est introuvable.", http.StatusNotFound)
		return
	}
	log.Printf("%s DB error: %v", operation, err)
	http.Error(w, "Impossible d’accéder aux données demandées.", http.StatusInternalServerError)
}
