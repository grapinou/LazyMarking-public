package marking

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

func TestHybridPendingDisagreementCannotDownloadArtifacts(t *testing.T) {
	for _, policy := range []string{"detector-agreement-v1", "detector-color-confidence-v1"} {
		t.Run(policy, func(t *testing.T) {
			fixture := newReviewPageFixture(t)
			if _, err := fixture.conn.Exec(`
		UPDATE marking_jobs
		SET review_policy_version=?, exam_name='corrected.pdf', mark_table_name='mark-table.pdf'
		WHERE id=50;
		UPDATE marking_answer_detections
		SET review_reason='detector_disagreement'
		WHERE id IN (700,701);
	`, policy); err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "/dashboard/marking/pdf?operation=marking-50&file=corrected.pdf", nil)
			request.AddCookie(markingSessionCookie(t, request))
			response := httptest.NewRecorder()
			login.CheckAuth(tools.HandlerWithDB(ServeFullMarkingPdfHandler, fixture.queries)).ServeHTTP(response, request)
			if response.Code != http.StatusConflict {
				t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
			}
		})
	}
}

func TestHybridLastDisagreementReviewRegeneratesAndFinalizes(t *testing.T) {
	for _, policy := range []string{"detector-agreement-v1", "detector-color-confidence-v1"} {
		t.Run(policy, func(t *testing.T) {
			fixture := newReviewPageFixture(t)
			if _, err := fixture.conn.Exec(`
		UPDATE marking_jobs SET review_policy_version=? WHERE id=50;
		UPDATE marking_answer_detections SET review_reason='detector_disagreement' WHERE id IN (700,701);
	`, policy); err != nil {
				t.Fatal(err)
			}
			if _, err := db.ApplyMarkingAnswerReview(t.Context(), fixture.queries, db.ApplyMarkingAnswerReviewInput{
				UserID: 1, MarkingJobID: 50, AnswerDetectionID: 700, ReviewedState: 0, ExpectedJobReviewRevision: 1,
			}); err != nil {
				t.Fatal(err)
			}
			regenerated := false
			stubMarkingArtifactsRegeneration(t, func(ctx context.Context, queries *db.Queries, userID int64, username string, jobID int64) (tools.MarkingArtifactsRegenerationResult, error) {
				regenerated = true
				rows, err := queries.AdvanceMarkingArtifactsRevision(ctx, db.AdvanceMarkingArtifactsRevisionParams{MarkingJobID: jobID, UserID: userID, ExpectedReviewRevision: 3, ExpectedArtifactsRevision: 0})
				if err != nil || rows != 1 {
					t.Fatalf("advance rows=%d err=%v", rows, err)
				}
				return tools.MarkingArtifactsRegenerationResult{Regenerated: true, ReviewRevision: 3, ArtifactsRevision: 3}, nil
			})
			response := fixture.post(t, reviewForm("50", "701", "1", "2", ""))
			if response.Code != http.StatusSeeOther || !regenerated {
				t.Fatalf("status=%d regenerated=%v", response.Code, regenerated)
			}
			summary, err := fixture.queries.GetMarkingReviewSummary(t.Context(), db.GetMarkingReviewSummaryParams{MarkingJobID: 50, UserID: 1})
			if err != nil || summary.PendingCandidates != 0 || summary.ReviewedCandidates != 2 {
				t.Fatalf("summary=%+v err=%v", summary, err)
			}
			var reviewRevision, artifactsRevision int64
			if err := fixture.conn.QueryRow(`SELECT review_revision,artifacts_revision FROM marking_jobs WHERE id=50`).Scan(&reviewRevision, &artifactsRevision); err != nil || reviewRevision != 3 || artifactsRevision != 3 {
				t.Fatalf("revisions=(%d,%d) err=%v", reviewRevision, artifactsRevision, err)
			}
		})
	}
}
