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
		http.Error(w, "Not found", http.StatusNotFound)
	default:
		log.Printf("%s integrity anomaly: affected %d rows", operation, rows)
		http.Error(w, "DB error", http.StatusInternalServerError)
	}
	return false
}

func HandleOwnedLookupError(w http.ResponseWriter, err error, operation string) {
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	log.Printf("%s DB error: %v", operation, err)
	http.Error(w, "DB error", http.StatusInternalServerError)
}
