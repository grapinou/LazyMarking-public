package exams

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
)

func setupExamHandlerTest(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	_, err = conn.Exec(`
CREATE TABLE qcm(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
CREATE TABLE class_codes(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
CREATE TABLE periods(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
CREATE TABLE years(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
CREATE TABLE exams(id INTEGER PRIMARY KEY, name TEXT NOT NULL, qcm_id INTEGER NOT NULL, class_code_id INTEGER NOT NULL, period_id INTEGER NOT NULL, year_id INTEGER NOT NULL, user_id INTEGER NOT NULL, UNIQUE(name,qcm_id,class_code_id,user_id));
CREATE TABLE exams_generated(id INTEGER PRIMARY KEY, exam_id INTEGER NOT NULL REFERENCES exams(id) ON DELETE RESTRICT, processed_students INTEGER NOT NULL DEFAULT 0, total_students INTEGER NOT NULL, status TEXT NOT NULL DEFAULT 'running', user_id INTEGER NOT NULL, UNIQUE(exam_id,user_id));
INSERT INTO qcm VALUES (1,'q1',1),(2,'q2',2);
INSERT INTO class_codes VALUES (1,'c1',1),(2,'c2',2);
INSERT INTO periods VALUES (1,'p1',1),(2,'p2',2);
INSERT INTO years VALUES (1,'y1',1),(2,'y2',2);
INSERT INTO exams VALUES (1,'owned',1,1,1,1,1),(2,'foreign',2,2,2,2,2);`)
	if err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func TestDeleteExamHandlerProtectsGeneratedHistory(t *testing.T) {
	for _, status := range []string{"running", "success", "failed"} {
		t.Run(status, func(t *testing.T) {
			conn, queries := setupExamHandlerTest(t)
			if _, err := conn.Exec("INSERT INTO exams_generated(id,exam_id,total_students,status,user_id) VALUES(10,1,1,?,1)", status); err != nil {
				t.Fatal(err)
			}

			response := serveAuthenticatedExamRequest(t, http.MethodPost, "/", url.Values{"exam_id": {"1"}}, func(w http.ResponseWriter, r *http.Request) {
				DeleteExamHandler(w, r, queries)
			})
			if response.Code != http.StatusSeeOther {
				t.Fatalf("status=%d, want 303", response.Code)
			}
			if location := response.Header().Get("Location"); !strings.HasPrefix(location, "/dashboard/errorsmessages?errormessage=") {
				t.Fatalf("Location=%q, want generated-exam business error", location)
			}
			assertExamAndGenerationCount(t, conn, 1, 1)
		})
	}
}

func TestDeleteExamHandlerTranslatesGenerationCreatedAfterPrecheck(t *testing.T) {
	conn, queries := setupExamHandlerTest(t)
	previous := afterExamDeletePrecheck
	afterExamDeletePrecheck = func(ctx context.Context, q *db.Queries, examID, userID int64) error {
		_, err := q.CreateExamGenerated(ctx, db.CreateExamGeneratedParams{ExamID: examID, TotalStudents: 1, UserID: userID})
		return err
	}
	t.Cleanup(func() { afterExamDeletePrecheck = previous })

	response := serveAuthenticatedExamRequest(t, http.MethodPost, "/", url.Values{"exam_id": {"1"}}, func(w http.ResponseWriter, r *http.Request) {
		DeleteExamHandler(w, r, queries)
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want 303", response.Code)
	}
	if location := response.Header().Get("Location"); !strings.HasPrefix(location, "/dashboard/errorsmessages?errormessage=") {
		t.Fatalf("Location=%q, want generated-exam business error", location)
	}
	assertExamAndGenerationCount(t, conn, 1, 1)
}

func TestDeleteExamHandlerReturnsInternalServerErrorForNonForeignKeyDBError(t *testing.T) {
	conn, queries := setupExamHandlerTest(t)
	previous := afterExamDeletePrecheck
	afterExamDeletePrecheck = func(ctx context.Context, _ *db.Queries, _, _ int64) error {
		_, err := conn.ExecContext(ctx, "DROP TABLE exams")
		return err
	}
	t.Cleanup(func() { afterExamDeletePrecheck = previous })

	response := serveAuthenticatedExamRequest(t, http.MethodPost, "/", url.Values{"exam_id": {"1"}}, func(w http.ResponseWriter, r *http.Request) {
		DeleteExamHandler(w, r, queries)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", response.Code)
	}
}

func assertExamAndGenerationCount(t *testing.T, conn *sql.DB, examCount, generationCount int) {
	t.Helper()
	var got int
	if err := conn.QueryRow("SELECT count(*) FROM exams WHERE id=1").Scan(&got); err != nil || got != examCount {
		t.Fatalf("exam count=%d err=%v, want %d", got, err, examCount)
	}
	if err := conn.QueryRow("SELECT count(*) FROM exams_generated WHERE exam_id=1").Scan(&got); err != nil || got != generationCount {
		t.Fatalf("generation count=%d err=%v, want %d", got, err, generationCount)
	}
}

func TestExamMutationHandlersDoNotReportSuccessForMissingOrForeignRows(t *testing.T) {
	conn, queries := setupExamHandlerTest(t)
	tests := []struct {
		name  string
		form  url.Values
		serve http.HandlerFunc
	}{
		{name: "create with foreign QCM", form: examForm("new", "1", "2", "1", "1", "1"), serve: func(w http.ResponseWriter, r *http.Request) { AddExamHandler(w, r, queries) }},
		{name: "create with foreign class", form: examForm("new", "1", "1", "2", "1", "1"), serve: func(w http.ResponseWriter, r *http.Request) { AddExamHandler(w, r, queries) }},
		{name: "create with foreign period", form: examForm("new", "1", "1", "1", "2", "1"), serve: func(w http.ResponseWriter, r *http.Request) { AddExamHandler(w, r, queries) }},
		{name: "create with foreign year", form: examForm("new", "1", "1", "1", "1", "2"), serve: func(w http.ResponseWriter, r *http.Request) { AddExamHandler(w, r, queries) }},
		{name: "update missing", form: examForm("new", "999", "1", "1", "1", "1"), serve: func(w http.ResponseWriter, r *http.Request) { EditExamHandler(w, r, queries) }},
		{name: "update foreign target", form: examForm("new", "2", "1", "1", "1", "1"), serve: func(w http.ResponseWriter, r *http.Request) { EditExamHandler(w, r, queries) }},
		{name: "update with foreign QCM", form: examForm("new", "1", "2", "1", "1", "1"), serve: func(w http.ResponseWriter, r *http.Request) { EditExamHandler(w, r, queries) }},
		{name: "update with foreign class", form: examForm("new", "1", "1", "2", "1", "1"), serve: func(w http.ResponseWriter, r *http.Request) { EditExamHandler(w, r, queries) }},
		{name: "update with foreign period", form: examForm("new", "1", "1", "1", "2", "1"), serve: func(w http.ResponseWriter, r *http.Request) { EditExamHandler(w, r, queries) }},
		{name: "update with foreign year", form: examForm("new", "1", "1", "1", "1", "2"), serve: func(w http.ResponseWriter, r *http.Request) { EditExamHandler(w, r, queries) }},
		{name: "delete missing", form: url.Values{"exam_id": {"999"}}, serve: func(w http.ResponseWriter, r *http.Request) { DeleteExamHandler(w, r, queries) }},
		{name: "delete foreign", form: url.Values{"exam_id": {"2"}}, serve: func(w http.ResponseWriter, r *http.Request) { DeleteExamHandler(w, r, queries) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedExamRequest(t, http.MethodPost, "/", tc.form, tc.serve)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want 404", response.Code)
			}
		})
	}
	var name string
	if err := conn.QueryRow("SELECT name FROM exams WHERE id=1 AND user_id=1").Scan(&name); err != nil || name != "owned" {
		t.Fatalf("owned exam name=%q after rejected updates, err=%v", name, err)
	}
	if err := conn.QueryRow("SELECT name FROM exams WHERE id=2 AND user_id=2").Scan(&name); err != nil || name != "foreign" {
		t.Fatalf("foreign exam name=%q err=%v", name, err)
	}
}

func TestExamFormsReturnNotFoundForMissingOrForeignExam(t *testing.T) {
	_, queries := setupExamHandlerTest(t)
	tests := []struct {
		name   string
		target string
		serve  http.HandlerFunc
	}{
		{name: "missing edit", target: "/?exam_id=999", serve: func(w http.ResponseWriter, r *http.Request) { EditFormExamHandler(w, r, queries) }},
		{name: "foreign edit", target: "/?exam_id=2", serve: func(w http.ResponseWriter, r *http.Request) { EditFormExamHandler(w, r, queries) }},
		{name: "missing delete", target: "/?exam_id=999", serve: func(w http.ResponseWriter, r *http.Request) { DeleteFormExamHandler(w, r, queries) }},
		{name: "foreign delete", target: "/?exam_id=2", serve: func(w http.ResponseWriter, r *http.Request) { DeleteFormExamHandler(w, r, queries) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedExamRequest(t, http.MethodGet, tc.target, nil, tc.serve)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want 404", response.Code)
			}
		})
	}
}

func TestOwnedExamMutationHandlersSucceed(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		conn, queries := setupExamHandlerTest(t)
		response := serveAuthenticatedExamRequest(t, http.MethodPost, "/", examForm("created", "", "1", "1", "1", "1"), func(w http.ResponseWriter, r *http.Request) {
			AddExamHandler(w, r, queries)
		})
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d, want 303", response.Code)
		}
		var count int
		if err := conn.QueryRow("SELECT count(*) FROM exams WHERE name='created' AND user_id=1").Scan(&count); err != nil || count != 1 {
			t.Fatalf("created exam count=%d err=%v", count, err)
		}
	})
	t.Run("update", func(t *testing.T) {
		conn, queries := setupExamHandlerTest(t)
		response := serveAuthenticatedExamRequest(t, http.MethodPost, "/", examForm("changed", "1", "1", "1", "1", "1"), func(w http.ResponseWriter, r *http.Request) {
			EditExamHandler(w, r, queries)
		})
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d, want 303", response.Code)
		}
		var name string
		if err := conn.QueryRow("SELECT name FROM exams WHERE id=1").Scan(&name); err != nil || name != "changed" {
			t.Fatalf("exam name=%q err=%v", name, err)
		}
	})
	t.Run("delete", func(t *testing.T) {
		conn, queries := setupExamHandlerTest(t)
		response := serveAuthenticatedExamRequest(t, http.MethodPost, "/", url.Values{"exam_id": {"1"}}, func(w http.ResponseWriter, r *http.Request) {
			DeleteExamHandler(w, r, queries)
		})
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d, want 303", response.Code)
		}
		var count int
		if err := conn.QueryRow("SELECT count(*) FROM exams WHERE id=1").Scan(&count); err != nil || count != 0 {
			t.Fatalf("deleted exam count=%d err=%v", count, err)
		}
	})
}

func examForm(name, examID, qcmID, classID, periodID, yearID string) url.Values {
	return url.Values{"exam": {name}, "exam_id": {examID}, "qcm_id": {qcmID}, "class_code_id": {classID}, "period_id": {periodID}, "year_id": {yearID}}
}

func serveAuthenticatedExamRequest(t *testing.T, method, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "exam-handler-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
	if form != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	session, err := login.GetStore().Get(request, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["user_id"] = int64(1)
	session.Values["username"] = "test-user"
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
