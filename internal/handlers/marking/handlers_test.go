package marking

import (
	"bytes"
	"context"
	"database/sql"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

func TestProcessingMarkingHandlerRequiresOwnedSuccessfulGeneration(t *testing.T) {
	t.Setenv("SESSION_KEY", "marking-upload-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())

	for _, tc := range []struct {
		name         string
		generationID string
		pages        int
		capacityFull bool
		wantStatus   int
		wantJob      bool
	}{
		{name: "owned success", generationID: "10", pages: 1, wantStatus: http.StatusSeeOther, wantJob: true},
		{name: "too many pages", generationID: "10", pages: MaxMarkingPDFPages + 1, wantStatus: http.StatusUnprocessableEntity},
		{name: "capacity full", generationID: "10", pages: 1, capacityFull: true, wantStatus: http.StatusServiceUnavailable},
		{name: "missing id", generationID: "", pages: 1, wantStatus: http.StatusBadRequest},
		{name: "invalid id", generationID: "abc", pages: 1, wantStatus: http.StatusBadRequest},
		{name: "foreign", generationID: "20", pages: 1, wantStatus: http.StatusNotFound},
		{name: "running", generationID: "11", pages: 1, wantStatus: http.StatusConflict},
		{name: "failed", generationID: "12", pages: 1, wantStatus: http.StatusConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn, err := sql.Open("sqlite3", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			conn.SetMaxOpenConns(1)
			defer conn.Close()
			if _, err := conn.Exec(`
				CREATE TABLE users(id INTEGER PRIMARY KEY, username TEXT NOT NULL);
				CREATE TABLE exams_generated(id INTEGER PRIMARY KEY, status TEXT NOT NULL, user_id INTEGER NOT NULL);
				CREATE TABLE marking_jobs(
					id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL,
					exam_generated_id INTEGER NOT NULL, total_pages INTEGER DEFAULT 0,
					done_pages INTEGER DEFAULT 0, total_exams INTEGER DEFAULT 0,
					done_exams INTEGER DEFAULT 0, status TEXT NOT NULL DEFAULT 'running',
					status_pdf TEXT NOT NULL DEFAULT 'running', exam_name TEXT,
					mark_table_name TEXT, completed_at TIMESTAMP,
					result_schema_version INTEGER, marking_algorithm_version TEXT,
					detection_threshold REAL, ambiguity_delta REAL,
					review_policy_version TEXT, v2_roi_radius_ratio REAL,
					v2_dark_pixel_threshold REAL, v2_dark_ratio_threshold REAL,
					v2_chroma_pixel_threshold REAL, v2_chroma_ratio_threshold REAL
				);
				INSERT INTO users VALUES (1, 'alice'), (2, 'bob');
				INSERT INTO exams_generated VALUES
					(10, 'success', 1), (11, 'running', 1), (12, 'failed', 1), (20, 'success', 2);
			`); err != nil {
				t.Fatal(err)
			}

			var releases []func()
			if tc.capacityFull {
				for range MaxConcurrentMarkingJobs {
					release, admitted := globalMarkingJobAdmission.tryAcquire()
					if !admitted {
						t.Fatal("fill marking capacity")
					}
					releases = append(releases, release)
				}
				defer func() {
					for _, release := range releases {
						release()
					}
				}()
			}

			request := markingUploadRequest(t, tc.generationID, tc.pages)
			request.AddCookie(markingSessionCookie(t, request))
			response := httptest.NewRecorder()
			var jobs sync.WaitGroup
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ProcessingMarkingHandler(w, r, db.New(conn), context.Background(), &jobs)
			})
			login.CheckAuth(handler).ServeHTTP(response, request)
			jobs.Wait()

			if response.Code != tc.wantStatus {
				t.Fatalf("status=%d body=%q, want %d", response.Code, response.Body.String(), tc.wantStatus)
			}
			var count int
			if err := conn.QueryRow("SELECT COUNT(*) FROM marking_jobs").Scan(&count); err != nil {
				t.Fatal(err)
			}
			wantCount := 0
			if tc.wantJob {
				wantCount = 1
			}
			if count != wantCount {
				t.Fatalf("job count=%d, want %d", count, wantCount)
			}
			if tc.wantJob {
				var generation, schemaVersion int64
				var algorithm string
				var threshold, ambiguityDelta float64
				if err := conn.QueryRow("SELECT exam_generated_id, result_schema_version, marking_algorithm_version, detection_threshold, ambiguity_delta FROM marking_jobs").Scan(&generation, &schemaVersion, &algorithm, &threshold, &ambiguityDelta); err != nil {
					t.Fatal(err)
				}
				if generation != 10 || schemaVersion != tools.MarkingResultSchemaVersion || algorithm != tools.MarkingAlgorithmVersion || threshold != tools.MarkingDetectionThreshold || ambiguityDelta != 0 {
					t.Fatalf("metadata=(%d,%d,%q,%v,%v), want new-format constants", generation, schemaVersion, algorithm, threshold, ambiguityDelta)
				}
			}
		})
	}
}

func markingUploadRequest(t *testing.T, generationID string, pages int) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if generationID != "" {
		if err := writer.WriteField("exam_generated_id", generationID); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("pdffile", "copies-"+strconv.Itoa(len(generationID))+".pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(syntheticMarkingPDF(pages, 595, 842)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/dashboard/marking/processing", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func TestMarkingResultPagesReturnNotFoundForMissingOwnedJob(t *testing.T) {
	t.Setenv("SESSION_KEY", "marking-handler-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}

	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`CREATE TABLE marking_jobs (
		id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL,
		status TEXT NOT NULL, status_pdf TEXT NOT NULL,
		total_pages INTEGER, done_pages INTEGER,
		total_exams INTEGER, done_exams INTEGER,
		exam_name TEXT, mark_table_name TEXT
	)`); err != nil {
		t.Fatal(err)
	}
	queries := db.New(conn)

	for _, tc := range []struct {
		name    string
		path    string
		handler func(http.ResponseWriter, *http.Request, *db.Queries)
	}{
		{name: "progress", path: "/progress?job_id=999", handler: ProgressMarkingHandler},
		{name: "success", path: "/success?job_id=999", handler: SuccessMarkingProcessingHandler},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tc.path, nil)
			request.AddCookie(markingSessionCookie(t, request))
			response := httptest.NewRecorder()
			login.CheckAuth(tools.HandlerWithDB(tc.handler, queries)).ServeHTTP(response, request)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d body=%q, want 404", response.Code, response.Body.String())
			}
		})
	}
}

func TestSuccessMarkingProcessingHandlerHidesForeignAndNonSuccessfulJobs(t *testing.T) {
	t.Setenv("SESSION_KEY", "marking-result-boundary-test-key-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	defer conn.Close()
	if _, err := conn.Exec(`
		CREATE TABLE marking_jobs (
			id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL,
			status TEXT NOT NULL, status_pdf TEXT NOT NULL,
			total_pages INTEGER, done_pages INTEGER, total_exams INTEGER, done_exams INTEGER,
			exam_name TEXT, mark_table_name TEXT
		);
		INSERT INTO marking_jobs(id,user_id,status,status_pdf) VALUES
			(10,1,'running','running'), (11,1,'failed','failed'), (12,2,'success','success');
	`); err != nil {
		t.Fatal(err)
	}
	handler := login.CheckAuth(tools.HandlerWithDB(SuccessMarkingProcessingHandler, db.New(conn)))
	for _, jobID := range []int{10, 11, 12, 999} {
		request := httptest.NewRequest(http.MethodGet, "/dashboard/marking/success?job_id="+strconv.Itoa(jobID), nil)
		request.AddCookie(markingSessionCookie(t, request))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("job %d status=%d body=%q, want 404", jobID, response.Code, response.Body.String())
		}
	}
}

func TestProgressMarkingFailedPollingIsStableAndNonDestructive(t *testing.T) {
	t.Setenv("SESSION_KEY", "marking-failed-poll-test-key-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	defer conn.Close()
	if _, err := conn.Exec(`
		CREATE TABLE marking_jobs (
			id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL,
			status TEXT NOT NULL, status_pdf TEXT NOT NULL,
			total_pages INTEGER, done_pages INTEGER,
			total_exams INTEGER, done_exams INTEGER,
			exam_name TEXT, mark_table_name TEXT
		);
		INSERT INTO marking_jobs(id, user_id, status, status_pdf) VALUES (42, 1, 'failed', 'running');
	`); err != nil {
		t.Fatal(err)
	}
	handler := login.CheckAuth(tools.HandlerWithDB(ProgressMarkingHandler, db.New(conn)))
	for poll := 1; poll <= 2; poll++ {
		request := httptest.NewRequest(http.MethodGet, "/progress?job_id=42", nil)
		request.AddCookie(markingSessionCookie(t, request))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("poll %d status=%d body=%q, want 303", poll, response.Code, response.Body.String())
		}
		if location := response.Header().Get("Location"); location == "" {
			t.Fatalf("poll %d has no error redirect", poll)
		}
		var count int
		if err := conn.QueryRow("SELECT COUNT(*) FROM marking_jobs WHERE id = 42 AND status = 'failed'").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("poll %d deleted failed job", poll)
		}
	}
}

func markingSessionCookie(t *testing.T, request *http.Request) *http.Cookie {
	t.Helper()
	session, err := login.GetStore().Get(request, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["user_id"] = int64(1)
	session.Values["username"] = "alice"
	response := httptest.NewRecorder()
	if err := session.Save(request, response); err != nil {
		t.Fatal(err)
	}
	return response.Result().Cookies()[0]
}
