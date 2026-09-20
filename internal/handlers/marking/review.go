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
	requestedDetectionID := int64(0)
	if value := r.URL.Query().Get("answer_detection_id"); value != "" {
		requestedDetectionID, err = strconv.ParseInt(value, 10, 64)
		if err != nil || requestedDetectionID <= 0 {
			http.Error(w, "Requête invalide", http.StatusBadRequest)
			return
		}
	}
	revisit := r.URL.Query().Get("revisit") == "1" || requestedDetectionID > 0
	if status != db.MarkingReviewPending && !(status == db.MarkingReviewCompleted && revisit) {
		http.Redirect(w, r, resultURL, http.StatusSeeOther)
		return
	}
	allCandidates, err := queries.ListMarkingReviewCandidates(r.Context(), db.ListMarkingReviewCandidatesParams{MarkingJobID: jobID, UserID: userID})
	if err != nil {
		log.Printf("From MarkingReviewHandler -> ListMarkingReviewCandidates: %v", err)
		http.Error(w, "Une erreur est survenue", http.StatusInternalServerError)
		return
	}
	if len(allCandidates) == 0 {
		http.Redirect(w, r, resultURL, http.StatusSeeOther)
		return
	}
	selectedIndex := -1
	if requestedDetectionID > 0 {
		for index, candidate := range allCandidates {
			if candidate.AnswerDetectionID == requestedDetectionID {
				selectedIndex = index
				break
			}
		}
		if selectedIndex < 0 {
			http.NotFound(w, r)
			return
		}
	} else if status == db.MarkingReviewPending {
		for index, candidate := range allCandidates {
			if !candidate.ReviewedState.Valid {
				selectedIndex = index
				break
			}
		}
	} else {
		selectedIndex = 0
	}
	if selectedIndex < 0 {
		http.Redirect(w, r, resultURL, http.StatusSeeOther)
		return
	}
	selected := allCandidates[selectedIndex]
	target, err := queries.GetMarkingAnswerReviewTarget(r.Context(), db.GetMarkingAnswerReviewTargetParams{
		MarkingJobID: jobID, UserID: userID, AnswerDetectionID: selected.AnswerDetectionID,
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
	candidate := markingReviewCandidate{
		AnswerDetectionID: selected.AnswerDetectionID,
		QuestionIndex:     selected.QuestionIndex,
		AnswerIndex:       selected.AnswerIndex,
		DetectedState:     selected.DetectedState,
		ReviewedState:     selected.ReviewedState,
	}
	if status == db.MarkingReviewCompleted {
		candidate.Position = int64(selectedIndex + 1)
	}
	page, err := buildMarkingReviewPageData(jobID, summary, candidate, target, resultURL)
	if err != nil {
		log.Printf("From MarkingReviewHandler -> build view data: %v", err)
		http.Error(w, "Une erreur est survenue", http.StatusInternalServerError)
		return
	}
	if selectedIndex > 0 {
		page.PreviousURL = markingReviewCandidateURL(jobID, allCandidates[selectedIndex-1].AnswerDetectionID)
	}
	if selectedIndex+1 < len(allCandidates) {
		page.NextURL = markingReviewCandidateURL(jobID, allCandidates[selectedIndex+1].AnswerDetectionID)
	}
	if r.URL.Query().Get("notice") == "conflict" {
		page.Notice = data.NoticeView{
			Title: "La page a été actualisée",
			Text:  "Cette correction a été modifiée dans un autre onglet. Vérifiez la réponse affichée avant de continuer.",
		}
	}
	RenderMarkingReviewPage(w, page)
}

type markingReviewCandidate struct {
	AnswerDetectionID int64
	QuestionIndex     int64
	AnswerIndex       int64
	DetectedState     int64
	ReviewedState     sql.NullInt64
	Position          int64
}

func buildMarkingReviewPageData(jobID int64, summary db.GetMarkingReviewSummaryRow, candidate markingReviewCandidate, target db.GetMarkingAnswerReviewTargetRow, resultURL string) (data.MarkingReviewPageData, error) {
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
	position := candidate.Position
	if position <= 0 {
		position = summary.ReviewedCandidates + 1
	}
	return data.MarkingReviewPageData{
		Routes: data.DefaultDashboardRoutes, MarkingRoutes: data.DefaultMarkingRoutes,
		PageTitle: "Vérification des réponses", JobID: jobID,
		Position: position, Total: summary.TotalCandidates, Remaining: summary.PendingCandidates,
		JobRevision: target.JobReviewRevision, AnswerReviewRevision: answerReviewRevision, ResultURL: resultURL,
		Candidate: data.MarkingReviewCandidateView{
			DetectionID: candidate.AnswerDetectionID, StudentDisplayName: studentName,
			QuestionNumber: candidate.QuestionIndex + 1, AnswerLabel: answerLabel,
			DetectedChecked: candidate.DetectedState == 1,
			HasReview:       candidate.ReviewedState.Valid, ReviewedChecked: candidate.ReviewedState.Int64 == 1,
			CropURL: cropURL,
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

func markingReviewCandidateURL(jobID, detectionID int64) string {
	return data.DefaultMarkingRoutes.ReviewURL + "?job_id=" + url.QueryEscape(strconv.FormatInt(jobID, 10)) +
		"&answer_detection_id=" + url.QueryEscape(strconv.FormatInt(detectionID, 10))
}
