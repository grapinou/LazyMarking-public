package studentclasscode

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestDeleteStudentClassCodeRejectsLastClass(t *testing.T) {
	conn, queries := newStudentClassHandlerTestDB(t)
	response := serveAuthenticatedStudentClassRequest(t, "/?student_id=1&class_code_id=10", func(w http.ResponseWriter, r *http.Request) {
		DeleteStudentClassCodeHandler(w, r, queries)
	})
	if response.Code != http.StatusSeeOther || !strings.HasPrefix(response.Header().Get("Location"), data.ErrorMessageURL+"?") {
		t.Fatalf("status=%d location=%q, want business redirect", response.Code, response.Header().Get("Location"))
	}
	assertStudentClassRelationCount(t, conn, 1, 10, 1)
	assertStudentClassTotal(t, conn, 1, 1)
}

func TestDeleteStudentClassCodeAllowsRemovalWhenAnotherClassRemains(t *testing.T) {
	conn, queries := newStudentClassHandlerTestDB(t)
	response := serveAuthenticatedStudentClassRequest(t, "/?student_id=2&class_code_id=10", func(w http.ResponseWriter, r *http.Request) {
		DeleteStudentClassCodeHandler(w, r, queries)
	})
	wantLocation := data.DefaultStudentRoutes.StudentClassCodesURL + "?student_id=2"
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != wantLocation {
		t.Fatalf("status=%d location=%q, want 303 to %q", response.Code, response.Header().Get("Location"), wantLocation)
	}
	assertStudentClassRelationCount(t, conn, 2, 10, 0)
	assertStudentClassRelationCount(t, conn, 2, 11, 1)
	assertStudentClassTotal(t, conn, 2, 1)
}

func TestDeleteStudentClassCodeRejectsMissingOrForeignContext(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{"missing student", "/?student_id=999&class_code_id=10"},
		{"foreign student", "/?student_id=3&class_code_id=10"},
		{"missing class", "/?student_id=1&class_code_id=999"},
		{"foreign class", "/?student_id=1&class_code_id=20"},
		{"missing relation", "/?student_id=1&class_code_id=11"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn, queries := newStudentClassHandlerTestDB(t)
			response := serveAuthenticatedStudentClassRequest(t, test.target, func(w http.ResponseWriter, r *http.Request) {
				DeleteStudentClassCodeHandler(w, r, queries)
			})
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want 404", response.Code)
			}
			assertStudentClassRelationCount(t, conn, 1, 10, 1)
			assertStudentClassTotal(t, conn, 1, 1)
		})
	}
}

func newStudentClassHandlerTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE students (id INTEGER PRIMARY KEY, first_name TEXT NOT NULL, last_name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE class_codes (id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE student_class_codes (
			id INTEGER PRIMARY KEY,
			student_id INTEGER NOT NULL,
			class_code_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			UNIQUE(student_id, class_code_id, user_id)
		);
		INSERT INTO students VALUES
			(1, 'Only', 'Class', 1),
			(2, 'Multi', 'Class', 1),
			(3, 'Foreign', 'Student', 2);
		INSERT INTO class_codes VALUES
			(10, 'Owned A', 1),
			(11, 'Owned B', 1),
			(20, 'Foreign', 2);
		INSERT INTO student_class_codes(student_id, class_code_id, user_id) VALUES
			(1, 10, 1),
			(2, 10, 1),
			(2, 11, 1),
			(3, 20, 2);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedStudentClassRequest(t *testing.T, target string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "student-class-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, target, nil)
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

func assertStudentClassRelationCount(t *testing.T, conn *sql.DB, studentID, classCodeID int64, want int) {
	t.Helper()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM student_class_codes WHERE student_id = ? AND class_code_id = ?", studentID, classCodeID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("relation (%d, %d) count=%d, want %d", studentID, classCodeID, count, want)
	}
}

func assertStudentClassTotal(t *testing.T, conn *sql.DB, studentID int64, want int) {
	t.Helper()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM student_class_codes WHERE student_id = ?", studentID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("student %d class count=%d, want %d", studentID, count, want)
	}
}
