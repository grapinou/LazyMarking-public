package marking

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestBuildMarkingResultPageDataReviewLifecycle(t *testing.T) {
	target := db.GetMarkingArtifactsRegenerationTargetRow{
		ReviewRevision: 2, ArtifactsRevision: 2,
		ExamName:      sql.NullString{String: "corrected.pdf", Valid: true},
		MarkTableName: sql.NullString{String: "mark-table.pdf", Valid: true},
	}
	tests := []struct {
		name          string
		status        db.MarkingReviewStatus
		total         int64
		pending       int64
		stale         bool
		wantTitle     string
		wantFinalPDFs bool
	}{
		{name: "no review needed", status: db.MarkingReviewNoReviewNeeded, wantTitle: "Aucune réponse à vérifier", wantFinalPDFs: true},
		{name: "pending", status: db.MarkingReviewPending, total: 3, pending: 2, wantTitle: "2 réponses à vérifier", wantFinalPDFs: true},
		{name: "completed current", status: db.MarkingReviewCompleted, total: 2, wantTitle: "Toutes les réponses ont été vérifiées", wantFinalPDFs: true},
		{name: "completed stale", status: db.MarkingReviewCompleted, total: 2, stale: true, wantTitle: "les PDF doivent être actualisés", wantFinalPDFs: false},
		{name: "legacy", status: db.MarkingReviewUnavailable, wantTitle: "Revue assistée non disponible", wantFinalPDFs: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			caseTarget := target
			if tc.stale {
				caseTarget.ArtifactsRevision = 1
			}
			page := buildMarkingResultPageData(42, caseTarget, db.GetMarkingReviewSummaryRow{
				TotalCandidates: tc.total, ReviewedCandidates: tc.total - tc.pending, PendingCandidates: tc.pending,
			}, tc.status, db.GetMarkingNonCorrectedSummaryRow{}, false)
			if !strings.Contains(page.Notice.Title, tc.wantTitle) {
				t.Fatalf("title=%q, want it to contain %q", page.Notice.Title, tc.wantTitle)
			}
			hasFinalPDFs := page.Artifacts.CorrectedPDFURL != "" && page.Artifacts.MarkTablePDFURL != ""
			if hasFinalPDFs != tc.wantFinalPDFs {
				t.Fatalf("final PDFs available=%v, want %v", hasFinalPDFs, tc.wantFinalPDFs)
			}
			if tc.status == db.MarkingReviewPending && !strings.Contains(page.Review.ReviewURL, "job_id=42") {
				t.Fatalf("review URL=%q", page.Review.ReviewURL)
			}
		})
	}
}

func TestBuildMarkingResultPageDataNonCorrectedIsIndependent(t *testing.T) {
	page := buildMarkingResultPageData(42, db.GetMarkingArtifactsRegenerationTargetRow{
		ReviewRevision: 3, ArtifactsRevision: 1,
		ExamName:      sql.NullString{String: "corrected.pdf", Valid: true},
		MarkTableName: sql.NullString{String: "mark-table.pdf", Valid: true},
	}, db.GetMarkingReviewSummaryRow{TotalCandidates: 1}, db.MarkingReviewCompleted,
		db.GetMarkingNonCorrectedSummaryRow{IncompleteCopies: 2, ErrorCopies: 1, NotSeenCopies: 3}, true)

	if page.Artifacts.CorrectedPDFURL != "" || page.Artifacts.MarkTablePDFURL != "" {
		t.Fatal("stale final PDFs must be hidden")
	}
	if !strings.Contains(page.Artifacts.NonCorrectedPDFURL, "corrected_NOT.pdf") {
		t.Fatalf("non-corrected PDF URL=%q", page.Artifacts.NonCorrectedPDFURL)
	}
	if page.NonCorrected.Total != 6 || page.NonCorrected.Incomplete != 2 || page.NonCorrected.Errors != 1 || page.NonCorrected.NotSeen != 3 {
		t.Fatalf("non-corrected summary=%+v", page.NonCorrected)
	}
}
