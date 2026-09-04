package marking

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

func TestBuildMarkingReviewPageDataUsesHistoricalSnapshotAndDetectedState(t *testing.T) {
	snapshot, _ := json.Marshal(config.QCM{Student: config.StudentQCM{FirstName: "Ada", LastName: "Lovelace"}})
	summary := db.GetMarkingReviewSummaryRow{TotalCandidates: 5, ReviewedCandidates: 2, PendingCandidates: 3}
	tests := []struct {
		name          string
		detectedState int64
		wantChecked   bool
	}{
		{name: "unchecked", detectedState: 0},
		{name: "checked", detectedState: 1, wantChecked: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			page, err := buildMarkingReviewPageData(42, summary, db.ListPendingMarkingReviewCandidatesRow{
				AnswerDetectionID: 77, QuestionIndex: 3, AnswerIndex: 1, DetectedState: tc.detectedState,
			}, db.GetMarkingAnswerReviewTargetRow{JobReviewRevision: 9, SnapshotContent: string(snapshot)}, "/result")
			if err != nil {
				t.Fatal(err)
			}
			if page.Position != 3 || page.Total != 5 || page.Remaining != 3 || page.JobRevision != 9 {
				t.Fatalf("progress=%+v", page)
			}
			if page.Candidate.StudentDisplayName != "Ada Lovelace" || page.Candidate.QuestionNumber != 4 || page.Candidate.AnswerLabel != "B" || page.Candidate.DetectedChecked != tc.wantChecked {
				t.Fatalf("candidate=%+v", page.Candidate)
			}
			if page.Candidate.CropURL != "/dashboard/marking/review/crop?job_id=42&answer_detection_id=77" {
				t.Fatalf("crop URL=%q", page.Candidate.CropURL)
			}
		})
	}
}

func TestBuildMarkingReviewPageDataCarriesOptionalAnswerRevision(t *testing.T) {
	snapshot, _ := json.Marshal(config.QCM{Student: config.StudentQCM{FirstName: "Ada", LastName: "Lovelace"}})
	page, err := buildMarkingReviewPageData(42,
		db.GetMarkingReviewSummaryRow{TotalCandidates: 1, PendingCandidates: 1},
		db.ListPendingMarkingReviewCandidatesRow{AnswerDetectionID: 77},
		db.GetMarkingAnswerReviewTargetRow{
			JobReviewRevision: 3, AnswerReviewRevision: sql.NullInt64{Int64: 2, Valid: true}, SnapshotContent: string(snapshot),
		}, "/result")
	if err != nil {
		t.Fatal(err)
	}
	if page.AnswerReviewRevision == nil || *page.AnswerReviewRevision != 2 {
		t.Fatalf("answer review revision=%v", page.AnswerReviewRevision)
	}
}

func TestMarkingAnswerLabelBeyondZ(t *testing.T) {
	for _, tc := range []struct {
		index int64
		want  string
	}{{0, "A"}, {25, "Z"}, {26, "AA"}, {27, "AB"}, {701, "ZZ"}, {702, "AAA"}} {
		got, err := markingAnswerLabel(tc.index)
		if err != nil || got != tc.want {
			t.Fatalf("label(%d)=%q err=%v, want %q", tc.index, got, err, tc.want)
		}
	}
	if _, err := markingAnswerLabel(-1); err == nil {
		t.Fatal("negative answer index accepted")
	}
}

func TestMarkingReviewHandlerPendingRendersFirstStableCandidate(t *testing.T) {
	fixture := newReviewPageFixture(t)
	for requestNumber := 0; requestNumber < 2; requestNumber++ {
		response := fixture.request(t, 50)
		if response.Code != http.StatusOK {
			t.Fatalf("request %d status=%d body=%q", requestNumber, response.Code, response.Body.String())
		}
		body := response.Body.String()
		for _, want := range []string{
			"Réponse 2 sur 3", "2 réponses restantes", "Ada Lovelace",
			"Question 1 — réponse A", "Détection automatique", "non cochée",
			"/dashboard/marking/review/crop?job_id=50&amp;answer_detection_id=701",
			"/dashboard/marking/success?job_id=50", "Extrait de la case à vérifier",
			`name="expected_review_revision" value="1"`, `name="expected_answer_review_revision" value=""`,
			`name="reviewed_state"`, `value="0"`, `value="1"`, "Valider et suivante",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("request %d body missing %q", requestNumber, want)
			}
		}
		for _, forbidden := range []string{"MeanGray", "mean_gray", "threshold", "ambiguity_delta", "Detection ID", "Job ID", "Excluded Copy"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("body exposes %q", forbidden)
			}
		}
		if strings.Contains(body, " checked") {
			t.Fatal("a human choice is preselected")
		}
	}
}

func TestApplyMarkingReviewHandlerConfirmationUsesPRGAndAdvances(t *testing.T) {
	fixture := newReviewPageFixture(t)
	beforeScore := fixture.copyScore(t, 500)
	response := fixture.post(t, reviewForm("50", "701", "0", "1", ""))
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/dashboard/marking/review?job_id=50" {
		t.Fatalf("status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
	if got := fixture.copyScore(t, 500); got != beforeScore {
		t.Fatalf("confirmation score=%d, want unchanged %d", got, beforeScore)
	}
	assertStoredReview(t, fixture.conn, 701, 0)

	next := fixture.request(t, 50)
	if next.Code != http.StatusOK || !strings.Contains(next.Body.String(), "Réponse 3 sur 3") || !strings.Contains(next.Body.String(), "Question 1 — réponse B") {
		t.Fatalf("next status=%d body=%q", next.Code, next.Body.String())
	}
	var beforeRefresh int
	if err := fixture.conn.QueryRow(`SELECT COUNT(*) FROM marking_answer_reviews`).Scan(&beforeRefresh); err != nil {
		t.Fatal(err)
	}
	refresh := fixture.request(t, 50)
	if refresh.Code != http.StatusOK {
		t.Fatalf("refresh status=%d", refresh.Code)
	}
	var afterRefresh int
	if err := fixture.conn.QueryRow(`SELECT COUNT(*) FROM marking_answer_reviews`).Scan(&afterRefresh); err != nil || afterRefresh != beforeRefresh {
		t.Fatalf("GET replayed mutation: before=%d after=%d err=%v", beforeRefresh, afterRefresh, err)
	}
}

func TestApplyMarkingReviewHandlerRespectsServiceIdempotence(t *testing.T) {
	fixture := newReviewPageFixture(t)
	first := fixture.post(t, reviewForm("50", "701", "0", "1", ""))
	if first.Code != http.StatusSeeOther {
		t.Fatalf("first status=%d body=%q", first.Code, first.Body.String())
	}
	second := fixture.post(t, reviewForm("50", "701", "0", "2", "1"))
	if second.Code != http.StatusSeeOther || second.Header().Get("Location") != "/dashboard/marking/review?job_id=50" {
		t.Fatalf("second status=%d location=%q body=%q", second.Code, second.Header().Get("Location"), second.Body.String())
	}
	var count, revision int64
	if err := fixture.conn.QueryRow(`SELECT COUNT(*),MAX(revision) FROM marking_answer_reviews WHERE answer_detection_id=701`).Scan(&count, &revision); err != nil || count != 1 || revision != 1 {
		t.Fatalf("review count=%d revision=%d err=%v", count, revision, err)
	}
}

func TestApplyMarkingReviewHandlerOverrideAndNonAmbiguousReview(t *testing.T) {
	t.Run("override pending", func(t *testing.T) {
		fixture := newReviewPageFixture(t)
		beforeScore := fixture.copyScore(t, 500)
		response := fixture.post(t, reviewForm("50", "701", "1", "1", ""))
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
		}
		assertStoredReview(t, fixture.conn, 701, 1)
		var effective int64
		if err := fixture.conn.QueryRow(`SELECT reviewed_state FROM marking_answer_reviews WHERE answer_detection_id=701`).Scan(&effective); err != nil || effective != 1 {
			t.Fatalf("effective review=%d err=%v", effective, err)
		}
		if afterScore := fixture.copyScore(t, 500); afterScore == beforeScore {
			t.Fatalf("override did not produce the service recalculation: score=%d", afterScore)
		}
	})
	t.Run("non ambiguous", func(t *testing.T) {
		fixture := newReviewPageFixture(t)
		response := fixture.post(t, reviewForm("50", "799", "0", "1", ""))
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
		}
		assertStoredReview(t, fixture.conn, 799, 0)
	})
}

func TestApplyMarkingReviewHandlerLastCandidateRegeneratesArtifacts(t *testing.T) {
	fixture := newReviewPageFixture(t)
	regenerationCalls := 0
	stubMarkingArtifactsRegeneration(t, func(ctx context.Context, queries *db.Queries, userID int64, username string, jobID int64) (tools.MarkingArtifactsRegenerationResult, error) {
		regenerationCalls++
		if userID != 1 || username != "alice" || jobID != 50 {
			t.Fatalf("regeneration scope=(%d,%q,%d)", userID, username, jobID)
		}
		rows, err := queries.AdvanceMarkingArtifactsRevision(ctx, db.AdvanceMarkingArtifactsRevisionParams{
			MarkingJobID: jobID, UserID: userID, ExpectedReviewRevision: 3, ExpectedArtifactsRevision: 0,
		})
		if err != nil || rows != 1 {
			t.Fatalf("advance artifacts rows=%d err=%v", rows, err)
		}
		return tools.MarkingArtifactsRegenerationResult{Regenerated: true, ReviewRevision: 3, ArtifactsRevision: 3}, nil
	})
	if _, err := db.ApplyMarkingAnswerReview(t.Context(), fixture.queries, db.ApplyMarkingAnswerReviewInput{
		UserID: 1, MarkingJobID: 50, AnswerDetectionID: 700, ReviewedState: 1,
		ExpectedJobReviewRevision: 1,
	}); err != nil {
		t.Fatal(err)
	}
	response := fixture.post(t, reviewForm("50", "701", "1", "2", ""))
	if response.Code != http.StatusSeeOther {
		t.Fatalf("POST status=%d body=%q", response.Code, response.Body.String())
	}
	if response.Header().Get("Location") != "/dashboard/marking/success?job_id=50" || regenerationCalls != 1 {
		t.Fatalf("location=%q regeneration calls=%d", response.Header().Get("Location"), regenerationCalls)
	}
	var reviewRevision, artifactsRevision int64
	if err := fixture.conn.QueryRow(`SELECT review_revision,artifacts_revision FROM marking_jobs WHERE id=50`).Scan(&reviewRevision, &artifactsRevision); err != nil || artifactsRevision != reviewRevision {
		t.Fatalf("revisions=(%d,%d) err=%v, want current", reviewRevision, artifactsRevision, err)
	}
}

func TestApplyMarkingReviewHandlerLastConfirmationFinalizes(t *testing.T) {
	fixture := newReviewPageFixture(t)
	if _, err := db.ApplyMarkingAnswerReview(t.Context(), fixture.queries, db.ApplyMarkingAnswerReviewInput{
		UserID: 1, MarkingJobID: 50, AnswerDetectionID: 700, ReviewedState: 1, ExpectedJobReviewRevision: 1,
	}); err != nil {
		t.Fatal(err)
	}
	called := false
	stubMarkingArtifactsRegeneration(t, func(context.Context, *db.Queries, int64, string, int64) (tools.MarkingArtifactsRegenerationResult, error) {
		called = true
		return tools.MarkingArtifactsRegenerationResult{}, nil
	})
	response := fixture.post(t, reviewForm("50", "701", "0", "2", ""))
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/dashboard/marking/success?job_id=50" || !called {
		t.Fatalf("status=%d location=%q called=%v", response.Code, response.Header().Get("Location"), called)
	}
	assertStoredReview(t, fixture.conn, 701, 0)
}

func TestApplyMarkingReviewHandlerDoesNotRegenerateBeforeLastCandidate(t *testing.T) {
	fixture := newReviewPageFixture(t)
	regenerationCalls := 0
	stubMarkingArtifactsRegeneration(t, func(context.Context, *db.Queries, int64, string, int64) (tools.MarkingArtifactsRegenerationResult, error) {
		regenerationCalls++
		return tools.MarkingArtifactsRegenerationResult{}, nil
	})
	response := fixture.post(t, reviewForm("50", "701", "0", "1", ""))
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/dashboard/marking/review?job_id=50" || regenerationCalls != 0 {
		t.Fatalf("status=%d location=%q regeneration calls=%d", response.Code, response.Header().Get("Location"), regenerationCalls)
	}
}

func TestApplyMarkingReviewHandlerRegenerationFailureKeepsCommittedReview(t *testing.T) {
	fixture := newReviewPageFixture(t)
	if _, err := db.ApplyMarkingAnswerReview(t.Context(), fixture.queries, db.ApplyMarkingAnswerReviewInput{
		UserID: 1, MarkingJobID: 50, AnswerDetectionID: 700, ReviewedState: 1, ExpectedJobReviewRevision: 1,
	}); err != nil {
		t.Fatal(err)
	}
	stubMarkingArtifactsRegeneration(t, func(context.Context, *db.Queries, int64, string, int64) (tools.MarkingArtifactsRegenerationResult, error) {
		return tools.MarkingArtifactsRegenerationResult{}, tools.ErrMarkingArtifactsRegeneration
	})
	response := fixture.post(t, reviewForm("50", "701", "1", "2", ""))
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/dashboard/marking/success?job_id=50&notice=artifacts_failed" {
		t.Fatalf("status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
	assertStoredReview(t, fixture.conn, 701, 1)
	var status string
	var reviewRevision, artifactsRevision int64
	if err := fixture.conn.QueryRow(`SELECT status,review_revision,artifacts_revision FROM marking_jobs WHERE id=50`).Scan(&status, &reviewRevision, &artifactsRevision); err != nil {
		t.Fatal(err)
	}
	if status != "success" || reviewRevision <= artifactsRevision {
		t.Fatalf("status=%q revisions=(%d,%d)", status, reviewRevision, artifactsRevision)
	}
}

func TestApplyMarkingReviewHandlerOptimisticConflictRefreshes(t *testing.T) {
	fixture := newReviewPageFixture(t)
	if _, err := db.ApplyMarkingAnswerReview(t.Context(), fixture.queries, db.ApplyMarkingAnswerReviewInput{
		UserID: 1, MarkingJobID: 50, AnswerDetectionID: 700, ReviewedState: 1,
		ExpectedJobReviewRevision: 1,
	}); err != nil {
		t.Fatal(err)
	}
	response := fixture.post(t, reviewForm("50", "701", "0", "1", ""))
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/dashboard/marking/review?job_id=50&notice=conflict" {
		t.Fatalf("status=%d location=%q body=%q", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	var count int
	if err := fixture.conn.QueryRow(`SELECT COUNT(*) FROM marking_answer_reviews WHERE answer_detection_id=701`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("stale decision written: count=%d err=%v", count, err)
	}
	notice := fixture.requestPath(t, response.Header().Get("Location"))
	if notice.Code != http.StatusOK || !strings.Contains(notice.Body.String(), "modifiée dans un autre onglet") {
		t.Fatalf("notice status=%d body=%q", notice.Code, notice.Body.String())
	}
}

func TestApplyMarkingReviewHandlerRejectsInvalidForm(t *testing.T) {
	valid := reviewForm("50", "701", "0", "1", "")
	tests := []struct {
		name, field, value string
		remove             bool
	}{
		{name: "missing state", field: "reviewed_state", remove: true},
		{name: "non numeric state", field: "reviewed_state", value: "x"},
		{name: "negative state", field: "reviewed_state", value: "-1"},
		{name: "state two", field: "reviewed_state", value: "2"},
		{name: "invalid job", field: "job_id", value: "x"},
		{name: "invalid detection", field: "answer_detection_id", value: "0"},
		{name: "invalid job revision", field: "expected_review_revision", value: "-1"},
		{name: "invalid answer revision zero", field: "expected_answer_review_revision", value: "0"},
		{name: "invalid answer revision", field: "expected_answer_review_revision", value: "x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newReviewPageFixture(t)
			form := cloneReviewForm(valid)
			if tc.remove {
				form.Del(tc.field)
			} else {
				form.Set(tc.field, tc.value)
			}
			response := fixture.post(t, form)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
			}
		})
	}
}

func TestApplyMarkingReviewHandlerRejectsOutOfScope(t *testing.T) {
	for _, tc := range []struct {
		name, job, detection string
	}{
		{name: "cross user", job: "51", detection: "720"},
		{name: "cross job", job: "50", detection: "720"},
		{name: "unknown", job: "50", detection: "999"},
		{name: "running", job: "52", detection: "701"},
		{name: "failed", job: "53", detection: "701"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newReviewPageFixture(t)
			response := fixture.post(t, reviewForm(tc.job, tc.detection, "1", "0", ""))
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
			}
		})
	}
}

func TestMarkingReviewHandlerDisplaysCheckedDetectedState(t *testing.T) {
	fixture := newReviewPageFixture(t)
	if _, err := fixture.conn.Exec(`INSERT INTO marking_answer_reviews(id,answer_detection_id,reviewer_user_id,reviewed_state,revision) VALUES(901,701,1,0,1)`); err != nil {
		t.Fatal(err)
	}
	response := fixture.request(t, 50)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "Question 1 — réponse B") || !strings.Contains(body, "<strong>cochée</strong>") || strings.Contains(body, "non cochée") {
		t.Fatalf("checked detection is not presented correctly: %q", body)
	}
}

func TestMarkingReviewHandlerOwnershipAndTechnicalLifecycle(t *testing.T) {
	fixture := newReviewPageFixture(t)
	for _, tc := range []struct {
		name  string
		jobID int64
	}{
		{name: "cross user", jobID: 51},
		{name: "running", jobID: 52},
		{name: "failed", jobID: 53},
		{name: "absent", jobID: 999},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := fixture.request(t, tc.jobID)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d body=%q, want 404", response.Code, response.Body.String())
			}
		})
	}
}

func TestMarkingReviewHandlerRedirectsWhenNoPendingQueue(t *testing.T) {
	fixture := newReviewPageFixture(t)
	for _, tc := range []struct {
		name  string
		jobID int64
	}{
		{name: "no review needed", jobID: 54},
		{name: "completed", jobID: 55},
		{name: "legacy", jobID: 56},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := fixture.request(t, tc.jobID)
			if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/dashboard/marking/success?job_id="+strconv.FormatInt(tc.jobID, 10) {
				t.Fatalf("status=%d location=%q", response.Code, response.Header().Get("Location"))
			}
		})
	}
}

type reviewPageFixture struct {
	conn    *sql.DB
	queries *db.Queries
}

func newReviewPageFixture(t *testing.T) reviewPageFixture {
	t.Helper()
	t.Setenv("SESSION_KEY", "marking-review-page-handler-test-key")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	t.Chdir("../../..")
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := conn.Exec(`
		CREATE TABLE marking_jobs(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,status TEXT NOT NULL,detection_threshold REAL,ambiguity_delta REAL,review_revision INTEGER NOT NULL,artifacts_revision INTEGER NOT NULL,review_policy_version TEXT,exam_name TEXT,mark_table_name TEXT);
		CREATE TABLE student_exam_content(student_exam_id INTEGER NOT NULL,user_id INTEGER NOT NULL,content TEXT NOT NULL);
		CREATE TABLE student_exam_page_content(student_exam_id INTEGER NOT NULL,page INTEGER NOT NULL,content TEXT NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE marking_copy_results(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,marking_job_id INTEGER NOT NULL,student_exam_id INTEGER NOT NULL,outcome TEXT NOT NULL,expected_pages INTEGER NOT NULL,detected_pages INTEGER NOT NULL,score_half_units INTEGER,total_points INTEGER,failure_code TEXT,failure_detail TEXT,completed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE marking_question_results(id INTEGER PRIMARY KEY,copy_result_id INTEGER NOT NULL,question_index INTEGER NOT NULL,state TEXT NOT NULL,score_half_units INTEGER NOT NULL,total_points INTEGER NOT NULL);
		CREATE TABLE marking_answer_detections(id INTEGER PRIMARY KEY,question_result_id INTEGER NOT NULL,answer_index INTEGER NOT NULL,detected_state INTEGER NOT NULL,mean_gray REAL NOT NULL,automatic_state INTEGER,review_reason TEXT);
		CREATE TABLE marking_answer_reviews(id INTEGER PRIMARY KEY,answer_detection_id INTEGER NOT NULL UNIQUE,reviewer_user_id INTEGER NOT NULL,reviewed_state INTEGER NOT NULL,reviewed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,revision INTEGER NOT NULL DEFAULT 1);
		CREATE TABLE marking_aligned_pages(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,copy_result_id INTEGER NOT NULL,page_exam INTEGER NOT NULL,storage_key TEXT NOT NULL);
		INSERT INTO marking_jobs(id,user_id,status,detection_threshold,ambiguity_delta,review_revision,artifacts_revision,review_policy_version) VALUES
			(50,1,'success',150,5,1,0,NULL),(51,2,'success',150,5,0,0,NULL),
			(52,1,'running',150,5,0,0,NULL),(53,1,'failed',150,5,0,0,NULL),
			(54,1,'success',150,5,0,0,NULL),(55,1,'success',150,5,1,0,NULL),(56,1,'success',NULL,NULL,0,0,NULL);
		INSERT INTO marking_copy_results(id,user_id,marking_job_id,student_exam_id,outcome,expected_pages,detected_pages,score_half_units,total_points) VALUES
			(500,1,50,100,'corrected',1,1,4,2),(501,1,50,101,'incomplete',1,0,NULL,NULL),
			(502,2,51,102,'corrected',1,1,0,1),(505,1,55,105,'corrected',1,1,0,1);
		INSERT INTO marking_question_results VALUES(600,500,0,'correct',2,1),(601,500,1,'correct',2,1),(610,501,0,'incorrect',0,1),(620,502,0,'incorrect',0,1),(650,505,0,'incorrect',0,1);
		INSERT INTO marking_answer_detections VALUES
			(700,600,1,1,149,NULL,NULL),(701,600,0,0,151,NULL,NULL),(702,601,0,1,150,NULL,NULL),
			(710,610,0,1,150,NULL,NULL),(720,620,0,1,150,NULL,NULL),(750,650,0,1,150,NULL,NULL),(799,600,2,1,220,NULL,NULL);
		INSERT INTO marking_answer_reviews(id,answer_detection_id,reviewer_user_id,reviewed_state,revision) VALUES(900,702,1,1,1),(950,750,1,1,1);
		INSERT INTO marking_aligned_pages VALUES(800,1,500,1,'aligned/student-exam-100/page-1.png'),(820,2,502,1,'aligned/student-exam-102/page-1.png'),(850,1,505,1,'aligned/student-exam-105/page-1.png');
	`); err != nil {
		t.Fatal(err)
	}
	snapshots := []struct {
		student, user int64
		first, last   string
	}{{100, 1, "Ada", "Lovelace"}, {101, 1, "Excluded", "Copy"}, {102, 2, "Foreign", "Student"}, {105, 1, "Reviewed", "Student"}}
	pageContent, _ := json.Marshal(config.PageContent{Questions: make([]config.CircleValidated, 2)})
	for _, item := range snapshots {
		questions := []config.Question{{Tags: config.Tags{Point: config.Point{PointValue: 1}}, Answers: []config.Answer{{State: 0}, {State: 1}, {State: 1}}}, {Tags: config.Tags{Point: config.Point{PointValue: 1}}, Answers: []config.Answer{{State: 1}}}}
		if item.student != 100 {
			questions = []config.Question{{Tags: config.Tags{Point: config.Point{PointValue: 1}}, Answers: []config.Answer{{State: 1}}}}
		}
		snapshot, _ := json.Marshal(config.QCM{Student: config.StudentQCM{FirstName: item.first, LastName: item.last}, Questions: questions})
		if _, err := conn.Exec(`INSERT INTO student_exam_content VALUES(?,?,?)`, item.student, item.user, string(snapshot)); err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(`INSERT INTO student_exam_page_content VALUES(?,1,?,?)`, item.student, string(pageContent), item.user); err != nil {
			t.Fatal(err)
		}
	}
	return reviewPageFixture{conn: conn, queries: db.New(conn)}
}

func (fixture reviewPageFixture) request(t *testing.T, jobID int64) *httptest.ResponseRecorder {
	t.Helper()
	return fixture.requestPath(t, "/dashboard/marking/review?job_id="+strconv.FormatInt(jobID, 10))
}

func (fixture reviewPageFixture) requestPath(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.AddCookie(markingSessionCookie(t, request))
	response := httptest.NewRecorder()
	login.CheckAuth(tools.HandlerWithDB(MarkingReviewHandler, fixture.queries)).ServeHTTP(response, request)
	return response
}

func (fixture reviewPageFixture) post(t *testing.T, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/dashboard/marking/review/apply", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(markingSessionCookie(t, request))
	response := httptest.NewRecorder()
	login.CheckAuth(tools.HandlerWithDB(ApplyMarkingReviewHandler, fixture.queries)).ServeHTTP(response, request)
	return response
}

func (fixture reviewPageFixture) copyScore(t *testing.T, copyID int64) int64 {
	t.Helper()
	var score int64
	if err := fixture.conn.QueryRow(`SELECT score_half_units FROM marking_copy_results WHERE id=?`, copyID).Scan(&score); err != nil {
		t.Fatal(err)
	}
	return score
}

func reviewForm(jobID, detectionID, state, jobRevision, answerRevision string) url.Values {
	return url.Values{
		"job_id":                          {jobID},
		"answer_detection_id":             {detectionID},
		"reviewed_state":                  {state},
		"expected_review_revision":        {jobRevision},
		"expected_answer_review_revision": {answerRevision},
	}
}

func cloneReviewForm(source url.Values) url.Values {
	clone := make(url.Values, len(source))
	for key, values := range source {
		clone[key] = append([]string(nil), values...)
	}
	return clone
}

func assertStoredReview(t *testing.T, conn *sql.DB, detectionID, wantState int64) {
	t.Helper()
	var state int64
	if err := conn.QueryRow(`SELECT reviewed_state FROM marking_answer_reviews WHERE answer_detection_id=?`, detectionID).Scan(&state); err != nil || state != wantState {
		t.Fatalf("review detection=%d state=%d err=%v, want %d", detectionID, state, err, wantState)
	}
}

func stubMarkingArtifactsRegeneration(t *testing.T, stub func(context.Context, *db.Queries, int64, string, int64) (tools.MarkingArtifactsRegenerationResult, error)) {
	t.Helper()
	previous := regenerateMarkingArtifacts
	regenerateMarkingArtifacts = stub
	t.Cleanup(func() { regenerateMarkingArtifacts = previous })
}
