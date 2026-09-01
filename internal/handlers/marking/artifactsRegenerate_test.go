package marking

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

func TestRegenerateMarkingArtifactsHandlerSuccessAndCurrentNoOp(t *testing.T) {
	fixture := newReviewPageFixture(t)
	for _, regenerated := range []bool{true, false} {
		t.Run(map[bool]string{true: "regenerated", false: "current no-op"}[regenerated], func(t *testing.T) {
			calls := 0
			stubMarkingArtifactsRegeneration(t, func(_ context.Context, _ *db.Queries, userID int64, username string, jobID int64) (tools.MarkingArtifactsRegenerationResult, error) {
				calls++
				if userID != 1 || username != "alice" || jobID != 50 {
					t.Fatalf("scope=(%d,%q,%d)", userID, username, jobID)
				}
				return tools.MarkingArtifactsRegenerationResult{Regenerated: regenerated}, nil
			})
			response := fixture.postRegeneration(t, "50")
			if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/dashboard/marking/success?job_id=50" || calls != 1 {
				t.Fatalf("status=%d location=%q calls=%d", response.Code, response.Header().Get("Location"), calls)
			}
		})
	}
}

func TestRegenerateMarkingArtifactsHandlerFailureIsRetryable(t *testing.T) {
	fixture := newReviewPageFixture(t)
	stubMarkingArtifactsRegeneration(t, func(context.Context, *db.Queries, int64, string, int64) (tools.MarkingArtifactsRegenerationResult, error) {
		return tools.MarkingArtifactsRegenerationResult{}, tools.ErrMarkingArtifactsConflict
	})
	response := fixture.postRegeneration(t, "50")
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/dashboard/marking/success?job_id=50&notice=artifacts_failed" {
		t.Fatalf("status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
}

func TestRegenerateMarkingArtifactsHandlerUnavailableIsNotFound(t *testing.T) {
	fixture := newReviewPageFixture(t)
	stubMarkingArtifactsRegeneration(t, func(context.Context, *db.Queries, int64, string, int64) (tools.MarkingArtifactsRegenerationResult, error) {
		return tools.MarkingArtifactsRegenerationResult{}, tools.ErrMarkingArtifactsUnavailable
	})
	for _, jobID := range []string{"51", "52", "53", "56", "999"} {
		if response := fixture.postRegeneration(t, jobID); response.Code != http.StatusNotFound {
			t.Fatalf("job=%s status=%d body=%q", jobID, response.Code, response.Body.String())
		}
	}
}

func TestRegenerateMarkingArtifactsHandlerRejectsInvalidJob(t *testing.T) {
	fixture := newReviewPageFixture(t)
	for _, jobID := range []string{"", "x", "0", "-1"} {
		if response := fixture.postRegeneration(t, jobID); response.Code != http.StatusBadRequest {
			t.Fatalf("job=%q status=%d", jobID, response.Code)
		}
	}
}

func (fixture reviewPageFixture) postRegeneration(t *testing.T, jobID string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"job_id": {jobID}}
	request := httptest.NewRequest(http.MethodPost, "/dashboard/marking/artifacts/regenerate", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(markingSessionCookie(t, request))
	response := httptest.NewRecorder()
	login.CheckAuth(tools.HandlerWithDB(RegenerateMarkingArtifactsHandler, fixture.queries)).ServeHTTP(response, request)
	return response
}
