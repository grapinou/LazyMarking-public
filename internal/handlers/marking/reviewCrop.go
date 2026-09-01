package marking

import (
	"errors"
	"log"
	"mime"
	"net/http"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

func MarkingReviewCropHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}
	jobID, err := strconv.ParseInt(r.URL.Query().Get("job_id"), 10, 64)
	if err != nil || jobID <= 0 {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}
	detectionID, err := strconv.ParseInt(r.URL.Query().Get("answer_detection_id"), 10, 64)
	if err != nil || detectionID <= 0 {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}
	roi, err := tools.ResolveMarkingReviewCandidateROI(r.Context(), queries, userID, username, jobID, detectionID)
	if errors.Is(err, tools.ErrMarkingReviewCandidateUnavailable) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("From MarkingReviewCropHandler -> resolve candidate: %v", err)
		http.Error(w, "Une erreur est survenue", http.StatusInternalServerError)
		return
	}
	crop, err := tools.BuildMarkingReviewCrop(roi)
	if errors.Is(err, tools.ErrMarkingReviewCandidateUnavailable) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("From MarkingReviewCropHandler -> build crop: %v", err)
		http.Error(w, "Une erreur est survenue", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": "review-crop.png"}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(crop); err != nil {
		log.Printf("From MarkingReviewCropHandler -> write crop: %v", err)
	}
}
