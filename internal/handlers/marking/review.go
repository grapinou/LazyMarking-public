package marking

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func MarkingReviewHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}
	jobID, err := strconv.ParseInt(r.URL.Query().Get("job_id"), 10, 64)
	if err != nil || jobID <= 0 {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}
	summary, err := queries.GetMarkingReviewSummary(r.Context(), db.GetMarkingReviewSummaryParams{MarkingJobID: jobID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("From MarkingReviewHandler -> GetMarkingReviewSummary: %v", err)
		http.Error(w, "Une erreur est survenue", http.StatusInternalServerError)
		return
	}
	status, err := db.DeriveMarkingReviewStatus(summary.AmbiguityDelta, summary.TotalCandidates, summary.PendingCandidates)
	if err != nil {
		log.Printf("From MarkingReviewHandler -> DeriveMarkingReviewStatus: %v", err)
		http.Error(w, "Une erreur est survenue", http.StatusInternalServerError)
		return
	}
	resultURL := markingResultURL(jobID)
	if status != db.MarkingReviewPending {
		http.Redirect(w, r, resultURL, http.StatusSeeOther)
		return
	}
	candidates, err := queries.ListPendingMarkingReviewCandidates(r.Context(), db.ListPendingMarkingReviewCandidatesParams{MarkingJobID: jobID, UserID: userID})
	if err != nil {
		log.Printf("From MarkingReviewHandler -> ListPendingMarkingReviewCandidates: %v", err)
		http.Error(w, "Une erreur est survenue", http.StatusInternalServerError)
		return
	}
	if len(candidates) == 0 {
		http.Redirect(w, r, resultURL, http.StatusSeeOther)
		return
	}
	first := candidates[0]
	target, err := queries.GetMarkingAnswerReviewTarget(r.Context(), db.GetMarkingAnswerReviewTargetParams{
		MarkingJobID: jobID, UserID: userID, AnswerDetectionID: first.AnswerDetectionID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("From MarkingReviewHandler -> GetMarkingAnswerReviewTarget: %v", err)
		http.Error(w, "Une erreur est survenue", http.StatusInternalServerError)
		return
	}
	page, err := buildMarkingReviewPageData(jobID, summary, first, target, resultURL)
	if err != nil {
		log.Printf("From MarkingReviewHandler -> build view data: %v", err)
		http.Error(w, "Une erreur est survenue", http.StatusInternalServerError)
		return
	}
	if r.URL.Query().Get("notice") == "conflict" {
		page.Notice = data.NoticeView{
			Title: "La page a été actualisée",
			Text:  "Cette correction a été modifiée dans un autre onglet. Vérifiez la réponse affichée avant de continuer.",
		}
	}
	RenderMarkingReviewPage(w, page)
}

func buildMarkingReviewPageData(jobID int64, summary db.GetMarkingReviewSummaryRow, candidate db.ListPendingMarkingReviewCandidatesRow, target db.GetMarkingAnswerReviewTargetRow, resultURL string) (data.MarkingReviewPageData, error) {
	var snapshot config.QCM
	if err := json.Unmarshal([]byte(target.SnapshotContent), &snapshot); err != nil {
		return data.MarkingReviewPageData{}, fmt.Errorf("decode student exam snapshot: %w", err)
	}
	studentName := strings.TrimSpace(snapshot.Student.FirstName + " " + snapshot.Student.LastName)
	if studentName == "" {
		studentName = "Élève"
	}
	answerLabel, err := markingAnswerLabel(candidate.AnswerIndex)
	if err != nil {
		return data.MarkingReviewPageData{}, err
	}
	cropURL := data.DefaultMarkingRoutes.ReviewCrop + "?job_id=" + url.QueryEscape(strconv.FormatInt(jobID, 10)) + "&answer_detection_id=" + url.QueryEscape(strconv.FormatInt(candidate.AnswerDetectionID, 10))
	var answerReviewRevision *int64
	if target.AnswerReviewRevision.Valid {
		revision := target.AnswerReviewRevision.Int64
		answerReviewRevision = &revision
	}
	return data.MarkingReviewPageData{
		Routes: data.DefaultDashboardRoutes, MarkingRoutes: data.DefaultMarkingRoutes,
		PageTitle: "Vérification des réponses", JobID: jobID,
		Position: summary.ReviewedCandidates + 1, Total: summary.TotalCandidates, Remaining: summary.PendingCandidates,
		JobRevision: target.JobReviewRevision, AnswerReviewRevision: answerReviewRevision, ResultURL: resultURL,
		Candidate: data.MarkingReviewCandidateView{
			DetectionID: candidate.AnswerDetectionID, StudentDisplayName: studentName,
			QuestionNumber: candidate.QuestionIndex + 1, AnswerLabel: answerLabel,
			DetectedChecked: candidate.DetectedState == 1, CropURL: cropURL,
		},
	}, nil
}

func markingAnswerLabel(answerIndex int64) (string, error) {
	if answerIndex < 0 {
		return "", errors.New("invalid answer index")
	}
	label := ""
	for index := answerIndex; ; index = index/26 - 1 {
		label = string(rune('A'+index%26)) + label
		if index < 26 {
			return label, nil
		}
	}
}

func markingResultURL(jobID int64) string {
	return data.DefaultMarkingRoutes.SuccessURL + "?job_id=" + url.QueryEscape(strconv.FormatInt(jobID, 10))
}
