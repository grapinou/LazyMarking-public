package marking

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	}
}

func TestMarkingReviewHandlerDisplaysCheckedDetectedState(t *testing.T) {
	fixture := newReviewPageFixture(t)
	if _, err := fixture.conn.Exec(`INSERT INTO marking_answer_reviews VALUES(901,701,0,0)`); err != nil {
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
		CREATE TABLE marking_jobs(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,status TEXT NOT NULL,detection_threshold REAL,ambiguity_delta REAL,review_revision INTEGER NOT NULL,artifacts_revision INTEGER NOT NULL);
		CREATE TABLE student_exam_content(student_exam_id INTEGER NOT NULL,user_id INTEGER NOT NULL,content TEXT NOT NULL);
		CREATE TABLE student_exam_page_content(student_exam_id INTEGER NOT NULL,page INTEGER NOT NULL,content TEXT NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE marking_copy_results(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,marking_job_id INTEGER NOT NULL,student_exam_id INTEGER NOT NULL,outcome TEXT NOT NULL);
		CREATE TABLE marking_question_results(id INTEGER PRIMARY KEY,copy_result_id INTEGER NOT NULL,question_index INTEGER NOT NULL,total_points INTEGER NOT NULL);
		CREATE TABLE marking_answer_detections(id INTEGER PRIMARY KEY,question_result_id INTEGER NOT NULL,answer_index INTEGER NOT NULL,detected_state INTEGER NOT NULL,mean_gray REAL NOT NULL);
		CREATE TABLE marking_answer_reviews(id INTEGER PRIMARY KEY,answer_detection_id INTEGER,reviewed_state INTEGER,revision INTEGER);
		CREATE TABLE marking_aligned_pages(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,copy_result_id INTEGER NOT NULL,page_exam INTEGER NOT NULL,storage_key TEXT NOT NULL);
		INSERT INTO marking_jobs VALUES
			(50,1,'success',150,5,1,0),(51,2,'success',150,5,0,0),
			(52,1,'running',150,5,0,0),(53,1,'failed',150,5,0,0),
			(54,1,'success',150,5,0,0),(55,1,'success',150,5,1,0),(56,1,'success',NULL,NULL,0,0);
		INSERT INTO marking_copy_results VALUES(500,1,50,100,'corrected'),(501,1,50,101,'incomplete'),(502,2,51,102,'corrected'),(505,1,55,105,'corrected');
		INSERT INTO marking_question_results VALUES(600,500,0,1),(601,500,1,1),(610,501,0,1),(620,502,0,1),(650,505,0,1);
		INSERT INTO marking_answer_detections VALUES
			(700,600,1,1,149),(701,600,0,0,151),(702,601,0,1,150),
			(710,610,0,1,150),(720,620,0,1,150),(750,650,0,1,150),(799,600,2,1,220);
		INSERT INTO marking_answer_reviews VALUES(900,702,1,0),(950,750,1,0);
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
		snapshot, _ := json.Marshal(config.QCM{Student: config.StudentQCM{FirstName: item.first, LastName: item.last}})
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
	request := httptest.NewRequest(http.MethodGet, "/dashboard/marking/review?job_id="+strconv.FormatInt(jobID, 10), nil)
	request.AddCookie(markingSessionCookie(t, request))
	response := httptest.NewRecorder()
	login.CheckAuth(tools.HandlerWithDB(MarkingReviewHandler, fixture.queries)).ServeHTTP(response, request)
	return response
}
