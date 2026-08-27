package students

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
)

func TestAddStudentRollsBackWhenClassOwnershipFails(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	form := url.Values{
		"first_name":    {"Should"},
		"last_name":     {"Rollback"},
		"class_code_id": {"20"},
	}
	response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
		AddStudentHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM students WHERE first_name = 'Should' AND last_name = 'Rollback'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("partially created students = %d, want 0", count)
	}
}

func TestStudentHandlersReturnNotFoundForMissingOrForeignStudent(t *testing.T) {
	_, queries := newStudentHandlerTestDB(t)
	tests := []struct {
		name   string
		method string
		target string
		form   url.Values
		serve  func(http.ResponseWriter, *http.Request)
	}{
		{name: "update missing", method: http.MethodPost, target: "/", form: url.Values{"student_id": {"999"}, "new_first_name": {"New"}, "new_last_name": {"Name"}}, serve: func(w http.ResponseWriter, r *http.Request) { EditStudentHandler(w, r, queries) }},
		{name: "delete foreign", method: http.MethodPost, target: "/", form: url.Values{"student_id": {"2"}}, serve: func(w http.ResponseWriter, r *http.Request) { DeleteStudentHandler(w, r, queries) }},
		{name: "edit form foreign", method: http.MethodGet, target: "/?student_id=2", serve: func(w http.ResponseWriter, r *http.Request) { EditFormStudentHandler(w, r, queries) }},
		{name: "delete form missing", method: http.MethodGet, target: "/?student_id=999", serve: func(w http.ResponseWriter, r *http.Request) { DeleteFormStudentHandler(w, r, queries) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedStudentRequest(t, tc.method, tc.target, tc.form, tc.serve)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
		})
	}
}

func TestDeleteAllStudentsRollsBackWhenMembershipRemovalFails(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	if _, err := conn.Exec(`
		INSERT INTO students(id, first_name, last_name, user_id) VALUES (3, 'Multi', 'Class', 1);
		INSERT INTO class_codes VALUES (11, 'Other owned', 1);
		INSERT INTO student_class_codes(student_id, class_code_id, user_id) VALUES (1, 10, 1), (3, 10, 1), (3, 11, 1);
		CREATE TRIGGER fail_membership_delete BEFORE DELETE ON student_class_codes
		BEGIN SELECT RAISE(ABORT, 'forced membership failure'); END;
	`); err != nil {
		t.Fatal(err)
	}
	response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", url.Values{"class_code_id": {"10"}}, func(w http.ResponseWriter, r *http.Request) {
		DeleteAllStudentsHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM students WHERE id = 1 AND user_id = 1").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("owned student count after rollback = %d, want 1", count)
	}
}

func newStudentHandlerTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE students (id INTEGER PRIMARY KEY AUTOINCREMENT, first_name TEXT NOT NULL, last_name TEXT NOT NULL, user_id INTEGER NOT NULL, UNIQUE(user_id, first_name, last_name));
		CREATE TABLE class_codes (id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE student_class_codes (id INTEGER PRIMARY KEY, student_id INTEGER NOT NULL, class_code_id INTEGER NOT NULL, user_id INTEGER NOT NULL, UNIQUE(student_id, class_code_id, user_id));
		INSERT INTO students(id, first_name, last_name, user_id) VALUES (1, 'Owned', 'Student', 1), (2, 'Foreign', 'Student', 2);
		INSERT INTO class_codes VALUES (10, 'Owned', 1), (20, 'Foreign', 2);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedStudentRequest(t *testing.T, method, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "student-handler-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	body := strings.NewReader(form.Encode())
	request := httptest.NewRequest(method, target, body)
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
