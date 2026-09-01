package marking

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

func TestMarkingReviewCropHandlerOwnerNonAmbiguousDetectionAndSecurityHeaders(t *testing.T) {
	fixture := newReviewCropHandlerFixture(t)
	response := fixture.request(t, 50, 700)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "image/png" || response.Header().Get("X-Content-Type-Options") != "nosniff" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("headers=%v", response.Header())
	}
	if _, err := png.Decode(bytes.NewReader(response.Body.Bytes())); err != nil {
		t.Fatalf("response is not a decodable PNG: %v", err)
	}
}

func TestMarkingReviewCropHandlerRejectsOutOfScopeTargets(t *testing.T) {
	fixture := newReviewCropHandlerFixture(t)
	for _, tc := range []struct {
		name        string
		jobID       int64
		detectionID int64
	}{
		{name: "unknown", jobID: 50, detectionID: 999},
		{name: "foreign job", jobID: 51, detectionID: 701},
		{name: "detection from another job", jobID: 50, detectionID: 701},
		{name: "running", jobID: 52, detectionID: 702},
		{name: "failed", jobID: 53, detectionID: 703},
		{name: "non corrected", jobID: 50, detectionID: 704},
		{name: "missing aligned page", jobID: 50, detectionID: 705},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := fixture.request(t, tc.jobID, tc.detectionID)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d body=%q, want 404", response.Code, response.Body.String())
			}
		})
	}
}

func TestMarkingReviewCropHandlerRejectsAlignedPageHashMismatch(t *testing.T) {
	fixture := newReviewCropHandlerFixture(t)
	if err := os.WriteFile(fixture.ownerPagePath, []byte("not a png"), 0o640); err != nil {
		t.Fatal(err)
	}
	response := fixture.request(t, 50, 700)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%q, want 404", response.Code, response.Body.String())
	}
}

func TestMarkingReviewCropHandlerRejectsCorruptPNG(t *testing.T) {
	fixture := newReviewCropHandlerFixture(t)
	corrupt := []byte("not a png")
	if err := os.WriteFile(fixture.ownerPagePath, corrupt, 0o640); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(corrupt)
	if _, err := fixture.conn.Exec(`UPDATE marking_aligned_pages SET sha256=? WHERE id=800`, hex.EncodeToString(digest[:])); err != nil {
		t.Fatal(err)
	}
	response := fixture.request(t, 50, 700)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%q, want 404", response.Code, response.Body.String())
	}
}

type reviewCropHandlerFixture struct {
	conn          *sql.DB
	queries       *db.Queries
	ownerPagePath string
}

func newReviewCropHandlerFixture(t *testing.T) reviewCropHandlerFixture {
	t.Helper()
	t.Setenv("SESSION_KEY", "marking-review-crop-handler-test-key")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY,username TEXT NOT NULL);
		CREATE TABLE marking_jobs(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,status TEXT NOT NULL,review_revision INTEGER NOT NULL,artifacts_revision INTEGER NOT NULL);
		CREATE TABLE student_exam_content(student_exam_id INTEGER NOT NULL,user_id INTEGER NOT NULL,content TEXT NOT NULL);
		CREATE TABLE student_exam_page_content(student_exam_id INTEGER NOT NULL,page INTEGER NOT NULL,content TEXT NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE marking_copy_results(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,marking_job_id INTEGER NOT NULL,student_exam_id INTEGER NOT NULL,outcome TEXT NOT NULL);
		CREATE TABLE marking_question_results(id INTEGER PRIMARY KEY,copy_result_id INTEGER NOT NULL,question_index INTEGER NOT NULL,total_points INTEGER NOT NULL);
		CREATE TABLE marking_answer_detections(id INTEGER PRIMARY KEY,question_result_id INTEGER NOT NULL,answer_index INTEGER NOT NULL,detected_state INTEGER NOT NULL,mean_gray REAL NOT NULL);
		CREATE TABLE marking_answer_reviews(id INTEGER PRIMARY KEY,answer_detection_id INTEGER,reviewed_state INTEGER,revision INTEGER);
		CREATE TABLE marking_aligned_pages(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,copy_result_id INTEGER NOT NULL,page_exam INTEGER NOT NULL,storage_key TEXT NOT NULL,width INTEGER NOT NULL,height INTEGER NOT NULL,sha256 TEXT NOT NULL,created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
		INSERT INTO users VALUES(1,'alice'),(2,'bob');
		INSERT INTO marking_jobs VALUES(50,1,'success',0,0),(51,2,'success',0,0),(52,1,'running',0,0),(53,1,'failed',0,0);
		INSERT INTO marking_copy_results VALUES
			(500,1,50,100,'corrected'),(501,2,51,101,'corrected'),(502,1,52,102,'corrected'),
			(503,1,53,103,'corrected'),(504,1,50,104,'incomplete'),(505,1,50,105,'corrected');
		INSERT INTO marking_question_results VALUES(600,500,0,1),(601,501,0,1),(602,502,0,1),(603,503,0,1),(604,504,0,1),(605,505,0,1);
		INSERT INTO marking_answer_detections VALUES(700,600,0,0,230),(701,601,0,0,230),(702,602,0,0,230),(703,603,0,0,230),(704,604,0,0,230),(705,605,0,0,230);
	`); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := json.Marshal(config.QCM{Questions: []config.Question{{Answers: []config.Answer{{}}}}})
	page, _ := json.Marshal(config.PageContent{Answers: []config.CircleValidated{{Position: config.Position{X: 40, Y: 40}, Radius: 12}}})
	for _, tc := range []struct{ student, user int64 }{{100, 1}, {101, 2}, {102, 1}, {103, 1}, {104, 1}, {105, 1}} {
		if _, err := conn.Exec(`INSERT INTO student_exam_content VALUES(?,?,?)`, tc.student, tc.user, string(snapshot)); err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(`INSERT INTO student_exam_page_content VALUES(?,1,?,?)`, tc.student, string(page), tc.user); err != nil {
			t.Fatal(err)
		}
	}
	ownerPath, ownerBytes := writeReviewCropHandlerPNG(t, "alice", 50, 100)
	digest := sha256.Sum256(ownerBytes)
	if _, err := conn.Exec(`INSERT INTO marking_aligned_pages VALUES(800,1,500,1,'aligned/student-exam-100/page-1.png',80,80,?,CURRENT_TIMESTAMP)`, hex.EncodeToString(digest[:])); err != nil {
		t.Fatal(err)
	}
	return reviewCropHandlerFixture{conn: conn, queries: db.New(conn), ownerPagePath: ownerPath}
}

func (fixture reviewCropHandlerFixture) request(t *testing.T, jobID, detectionID int64) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/dashboard/marking/review/crop?job_id="+strconv.FormatInt(jobID, 10)+"&answer_detection_id="+strconv.FormatInt(detectionID, 10), nil)
	request.AddCookie(markingSessionCookie(t, request))
	response := httptest.NewRecorder()
	login.CheckAuth(tools.HandlerWithDB(MarkingReviewCropHandler, fixture.queries)).ServeHTTP(response, request)
	return response
}

func writeReviewCropHandlerPNG(t *testing.T, username string, jobID, studentID int64) (string, []byte) {
	t.Helper()
	directory := filepath.Join("assets", "tmp", username, "marking-"+strconv.FormatInt(jobID, 10), "aligned", "student-exam-"+strconv.FormatInt(studentID, 10))
	if err := os.MkdirAll(directory, 0o750); err != nil {
		t.Fatal(err)
	}
	img := image.NewGray(image.Rect(0, 0, 80, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 80; x++ {
			img.SetGray(x, y, color.Gray{Y: 220})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "page-1.png")
	if err := os.WriteFile(path, encoded.Bytes(), 0o640); err != nil {
		t.Fatal(err)
	}
	return path, encoded.Bytes()
}
