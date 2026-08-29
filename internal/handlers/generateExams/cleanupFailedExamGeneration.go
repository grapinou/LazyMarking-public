package generateexams

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func cleanupFailedExamGeneration(userID, generationID int64, username string, ctx context.Context, queries *db.Queries) error {
	cleanupCtx := ctx
	cancel := func() {}
	if ctx.Err() != nil {
		cleanupCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	}
	defer cancel()

	status, err := queries.GetExamStatus(cleanupCtx, db.GetExamStatusParams{ID: generationID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return tools.CleanupFailedExamGeneration(cleanupCtx, queries, generationID, userID, username)
	}
	if err != nil {
		return fmt.Errorf("read generation %d before failed cleanup: %w", generationID, err)
	}
	if status == "success" {
		return fmt.Errorf("refuse failed cleanup of successful generation %d", generationID)
	}
	if status == "running" {
		if err := failExamGeneration(userID, generationID, cleanupCtx, queries); err != nil {
			if _, statusErr := queries.GetExamStatus(cleanupCtx, db.GetExamStatusParams{ID: generationID, UserID: userID}); errors.Is(statusErr, sql.ErrNoRows) {
				return tools.CleanupFailedExamGeneration(cleanupCtx, queries, generationID, userID, username)
			}
			return err
		}
	}
	return tools.CleanupFailedExamGeneration(cleanupCtx, queries, generationID, userID, username)
}

func handleCleanedFailedGenerationPoll(w http.ResponseWriter, r *http.Request, queries *db.Queries, userID int64) bool {
	if r.URL.Query().Get("generation_started") != "1" {
		return false
	}
	examID, err := strconv.ParseInt(r.URL.Query().Get("exam_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid exam parameter", http.StatusBadRequest)
		return true
	}
	if _, err := queries.GetExamByID(r.Context(), db.GetExamByIDParams{ID: examID, UserID: userID}); err != nil {
		tools.HandleOwnedLookupError(w, err, "GetExamProgressPageHandler cleaned generation GetExamByID")
		return true
	}
	redirectFailedExamGeneration(w, r)
	return true
}

func redirectFailedExamGeneration(w http.ResponseWriter, r *http.Request) {
	errorMessage := url.QueryEscape("La génération a échoué. Vous pouvez corriger l'évaluation puis réessayer.")
	http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
}
