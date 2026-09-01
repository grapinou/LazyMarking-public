package marking

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func ApplyMarkingReviewHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}
	jobID, ok := parsePositiveReviewFormInt(r.FormValue("job_id"))
	if !ok {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}
	detectionID, ok := parsePositiveReviewFormInt(r.FormValue("answer_detection_id"))
	if !ok {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}
	reviewedState, err := strconv.ParseInt(r.FormValue("reviewed_state"), 10, 64)
	if err != nil || (reviewedState != 0 && reviewedState != 1) {
		http.Error(w, "Choix invalide", http.StatusBadRequest)
		return
	}
	jobRevision, err := strconv.ParseInt(r.FormValue("expected_review_revision"), 10, 64)
	if err != nil || jobRevision < 0 {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}
	answerRevision, ok := parseOptionalAnswerReviewRevision(r.FormValue("expected_answer_review_revision"))
	if !ok {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}
	_, err = db.ApplyMarkingAnswerReview(r.Context(), queries, db.ApplyMarkingAnswerReviewInput{
		UserID: userID, MarkingJobID: jobID, AnswerDetectionID: detectionID,
		ReviewedState: reviewedState, ExpectedAnswerReviewRevision: answerRevision,
		ExpectedJobReviewRevision: jobRevision,
	})
	reviewURL := data.DefaultMarkingRoutes.ReviewURL + "?job_id=" + url.QueryEscape(strconv.FormatInt(jobID, 10))
	if errors.Is(err, db.ErrMarkingReviewUnavailable) {
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, db.ErrMarkingReviewConflict) {
		http.Redirect(w, r, reviewURL+"&notice=conflict", http.StatusSeeOther)
		return
	}
	if err != nil {
		log.Printf("From ApplyMarkingReviewHandler -> ApplyMarkingAnswerReview: %v", err)
		http.Error(w, "Une erreur est survenue", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, reviewURL, http.StatusSeeOther)
}

func parsePositiveReviewFormInt(value string) (int64, bool) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	return parsed, err == nil && parsed > 0
}

func parseOptionalAnswerReviewRevision(value string) (int64, bool) {
	if value == "" {
		return 0, true
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	return parsed, err == nil && parsed > 0
}
