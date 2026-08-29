package generateexams

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
)

func TestGenerateExamsHandlerRejectsEmptyQCMBeforeStartingGeneration(t *testing.T) {
	conn, queries := setupExamPreflightTest(t)
	t.Chdir(t.TempDir())
	var jobs sync.WaitGroup

	response := serveAuthenticatedGenerationRequest(t, "/dashboard/exams/generate?exam_id=1", func(w http.ResponseWriter, r *http.Request) {
		GenerateExamsHandler(w, r, queries, context.Background(), &jobs)
	})
	assertEmptyQCMRedirect(t, response)
	assertGenerationTablesEmpty(t, conn)
	if _, err := os.Stat("assets"); !os.IsNotExist(err) {
		t.Fatalf("generation workspace tree exists after rejected preflight: %v", err)
	}
	jobs.Wait()
}

func TestGenerateMiniPDFHandlerRejectsEmptyQCMBeforeWorkersOrWorkspace(t *testing.T) {
	conn, queries := setupExamPreflightTest(t)
	t.Chdir(t.TempDir())

	response := serveAuthenticatedGenerationRequest(t, "/dashboard/exams/generatemini?exam_id=1", func(w http.ResponseWriter, r *http.Request) {
		GenerateMiniPDFHandler(w, r, queries)
	})
	assertEmptyQCMRedirect(t, response)
	assertGenerationTablesEmpty(t, conn)
	if _, err := os.Stat("assets"); !os.IsNotExist(err) {
		t.Fatalf("mini workspace tree exists after rejected preflight: %v", err)
	}
}

func TestGenerateExamsHandlerRejectsEmptyClassBeforeGeneration(t *testing.T) {
	conn, queries := setupExamPreflightTest(t)
	var jobs sync.WaitGroup
	response := serveAuthenticatedGenerationRequest(t, "/dashboard/exams/generate?exam_id=4", func(w http.ResponseWriter, r *http.Request) {
		GenerateExamsHandler(w, r, queries, context.Background(), &jobs)
	})
	if response.Code != http.StatusSeeOther || !strings.Contains(response.Header().Get("Location"), "aucun+%C3%A9l%C3%A8ve") {
		t.Fatalf("status=%d Location=%q, want empty-class business redirect", response.Code, response.Header().Get("Location"))
	}
	assertGenerationTablesEmpty(t, conn)
}

func TestGenerateExamsHandlerOwnershipAndNonEmptyPreflight(t *testing.T) {
	tests := []struct {
		name       string
		examID     string
		wantStatus int
		wantText   string
	}{
		{name: "absent", examID: "999", wantStatus: http.StatusNotFound},
		{name: "foreign", examID: "3", wantStatus: http.StatusNotFound},
		{name: "incoherent foreign QCM", examID: "5", wantStatus: http.StatusNotFound},
		{name: "non-empty reaches existing-generation guard", examID: "2", wantStatus: http.StatusSeeOther, wantText: "d%C3%A9j%C3%A0+%C3%A9t%C3%A9+g%C3%A9n%C3%A9r%C3%A9"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, queries := setupExamPreflightTest(t)
			var jobs sync.WaitGroup
			response := serveAuthenticatedGenerationRequest(t, "/dashboard/exams/generate?exam_id="+tc.examID, func(w http.ResponseWriter, r *http.Request) {
				GenerateExamsHandler(w, r, queries, context.Background(), &jobs)
			})
			if response.Code != tc.wantStatus {
				t.Fatalf("status=%d, want %d", response.Code, tc.wantStatus)
			}
			if tc.wantText != "" && !strings.Contains(response.Header().Get("Location"), tc.wantText) {
				t.Fatalf("Location=%q, want %q", response.Header().Get("Location"), tc.wantText)
			}
		})
	}
}

func setupExamPreflightTest(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`
		CREATE TABLE qcm(id INTEGER PRIMARY KEY,name TEXT,user_id INTEGER);
		CREATE TABLE qcm_questions(id INTEGER PRIMARY KEY,qcm_id INTEGER,question_id INTEGER,position INTEGER,user_id INTEGER);
		CREATE TABLE class_codes(id INTEGER PRIMARY KEY,name TEXT,user_id INTEGER);
		CREATE TABLE periods(id INTEGER PRIMARY KEY,name TEXT,user_id INTEGER);
		CREATE TABLE years(id INTEGER PRIMARY KEY,name TEXT,user_id INTEGER);
		CREATE TABLE students(id INTEGER PRIMARY KEY,first_name TEXT,last_name TEXT,user_id INTEGER);
		CREATE TABLE student_class_codes(id INTEGER PRIMARY KEY,student_id INTEGER,class_code_id INTEGER,user_id INTEGER);
		CREATE TABLE exams(id INTEGER PRIMARY KEY,name TEXT,qcm_id INTEGER,class_code_id INTEGER,period_id INTEGER,year_id INTEGER,user_id INTEGER);
		CREATE TABLE exams_generated(id INTEGER PRIMARY KEY,exam_id INTEGER,processed_students INTEGER DEFAULT 0,total_students INTEGER,status TEXT DEFAULT 'running',user_id INTEGER,UNIQUE(exam_id,user_id));
		CREATE TABLE student_exam(id INTEGER PRIMARY KEY,exam_generated_id INTEGER,student_id INTEGER,user_id INTEGER);
		INSERT INTO qcm VALUES(1,'empty',1),(2,'ready',1),(3,'foreign',2);
		INSERT INTO qcm_questions VALUES(10,2,100,1,1);
		INSERT INTO class_codes VALUES(1,'1A',1),(3,'foreign',2),(4,'empty class',1);
		INSERT INTO periods VALUES(1,'P1',1),(3,'foreign',2);
		INSERT INTO years VALUES(1,'2026',1),(3,'foreign',2);
		INSERT INTO students VALUES(20,'Ada','Lovelace',1);
		INSERT INTO student_class_codes VALUES(30,20,1,1);
		INSERT INTO exams VALUES
			(1,'empty QCM',1,1,1,1,1),
			(2,'ready QCM',2,1,1,1,1),
			(3,'foreign Exam',3,3,3,3,2),
			(4,'empty class',2,4,1,1,1),
			(5,'legacy incoherent QCM',3,1,1,1,1);
		INSERT INTO exams_generated(id,exam_id,total_students,status,user_id) VALUES(99,2,1,'success',1);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedGenerationRequest(t *testing.T, target string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "exam-generation-preflight-key-32")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, target, nil)
	session, err := login.GetStore().Get(request, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["user_id"] = int64(1)
	session.Values["username"] = "teacher"
	cookies := httptest.NewRecorder()
	if err := session.Save(request, cookies); err != nil {
		t.Fatal(err)
	}
	for _, cookie := range cookies.Result().Cookies() {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	login.CheckAuth(handler).ServeHTTP(response, request)
	return response
}

func assertEmptyQCMRedirect(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want 303", response.Code)
	}
	location, err := url.QueryUnescape(response.Header().Get("Location"))
	if err != nil || !strings.Contains(location, "Ajoutez au moins une question au QCM") {
		t.Fatalf("Location=%q err=%v, want empty-QCM business message", location, err)
	}
}

func assertGenerationTablesEmpty(t *testing.T, conn *sql.DB) {
	t.Helper()
	for _, table := range []string{"student_exam"} {
		var count int
		if err := conn.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s count=%d err=%v, want 0", table, count, err)
		}
	}
	var count int
	if err := conn.QueryRow("SELECT count(*) FROM exams_generated WHERE exam_id IN (1,4)").Scan(&count); err != nil || count != 0 {
		t.Fatalf("new generation count=%d err=%v, want 0", count, err)
	}
}
