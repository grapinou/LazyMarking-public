package studentclasscode

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestDeleteFormStudentClassCodeDoesNotMutateAndRendersConfirmation(t *testing.T) {
	conn, queries := newStudentClassHandlerTestDB(t)
	withStudentClassRepositoryRoot(t, func() {
		response := serveAuthenticatedStudentClassRequest(t, http.MethodGet, "/?student_id=2&class_code_id=10", nil, func(w http.ResponseWriter, r *http.Request) {
			DeleteFormStudentClassCodeHandler(w, r, queries)
		})
		if response.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q, want 200", response.Code, response.Body.String())
		}
		for _, expected := range []string{
			"Multi Class", "Owned A", "method=\"post\"",
			"action=\"" + data.DefaultStudentClassCodeRoutes.DeleteURL + "\"",
			"name=\"student_id\" value=\"2\"", "name=\"class_code_id\" value=\"10\"",
			data.DefaultStudentRoutes.StudentClassCodesURL + "?student_id=2",
		} {
			if !strings.Contains(response.Body.String(), expected) {
				t.Errorf("body does not contain %q", expected)
			}
		}
	})
	assertStudentClassRelationCount(t, conn, 2, 10, 1)
	assertStudentClassTotal(t, conn, 2, 2)
}

func TestDeleteFormStudentClassCodeLastClassIsReadOnly(t *testing.T) {
	conn, queries := newStudentClassHandlerTestDB(t)
	withStudentClassRepositoryRoot(t, func() {
		response := serveAuthenticatedStudentClassRequest(t, http.MethodGet, "/?student_id=1&class_code_id=10", nil, func(w http.ResponseWriter, r *http.Request) {
			DeleteFormStudentClassCodeHandler(w, r, queries)
		})
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Impossible de retirer la dernière classe de l’élève.") {
			t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "action=\""+data.DefaultStudentClassCodeRoutes.DeleteURL+"\"") || strings.Contains(response.Body.String(), "Retirer de la classe</button>") {
			t.Fatal("last-class confirmation exposes an active destructive form")
		}
	})
	assertStudentClassRelationCount(t, conn, 1, 10, 1)
	assertStudentClassTotal(t, conn, 1, 1)
}

func TestDeleteStudentClassCodeRejectsLastClass(t *testing.T) {
	conn, queries := newStudentClassHandlerTestDB(t)
	response := serveAuthenticatedStudentClassRequest(t, http.MethodPost, "/", url.Values{"student_id": {"1"}, "class_code_id": {"10"}}, func(w http.ResponseWriter, r *http.Request) {
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
	response := serveAuthenticatedStudentClassRequest(t, http.MethodPost, "/", url.Values{"student_id": {"2"}, "class_code_id": {"10"}}, func(w http.ResponseWriter, r *http.Request) {
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
		values url.Values
	}{
		{"missing student", url.Values{"student_id": {"999"}, "class_code_id": {"10"}}},
		{"foreign student", url.Values{"student_id": {"3"}, "class_code_id": {"10"}}},
		{"missing class", url.Values{"student_id": {"1"}, "class_code_id": {"999"}}},
		{"foreign class", url.Values{"student_id": {"1"}, "class_code_id": {"20"}}},
		{"missing relation", url.Values{"student_id": {"1"}, "class_code_id": {"11"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn, queries := newStudentClassHandlerTestDB(t)
			response := serveAuthenticatedStudentClassRequest(t, http.MethodPost, "/", test.values, func(w http.ResponseWriter, r *http.Request) {
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

func TestDeleteFormStudentClassCodeRejectsMissingOrForeignContext(t *testing.T) {
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
			response := serveAuthenticatedStudentClassRequest(t, http.MethodGet, test.target, nil, func(w http.ResponseWriter, r *http.Request) {
				DeleteFormStudentClassCodeHandler(w, r, queries)
			})
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want 404", response.Code)
			}
			assertStudentClassRelationCount(t, conn, 1, 10, 1)
		})
	}
}

func TestDeleteStudentClassCodeRouteMethods(t *testing.T) {
	_, queries := newStudentClassHandlerTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, queries)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPut, data.DefaultStudentClassCodeRoutes.DeleteURL, nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d, want 405", response.Code)
	}
}

func TestDeleteStudentClassCodeRejectsInvalidParametersOnGetAndPost(t *testing.T) {
	_, queries := newStudentClassHandlerTestDB(t)
	tests := []struct {
		name   string
		method string
		target string
		values url.Values
		handle http.HandlerFunc
	}{
		{"GET missing parameter", http.MethodGet, "/?student_id=1", nil, func(w http.ResponseWriter, r *http.Request) { DeleteFormStudentClassCodeHandler(w, r, queries) }},
		{"GET invalid parameter", http.MethodGet, "/?student_id=nope&class_code_id=10", nil, func(w http.ResponseWriter, r *http.Request) { DeleteFormStudentClassCodeHandler(w, r, queries) }},
		{"POST missing parameter", http.MethodPost, "/", url.Values{"student_id": {"1"}}, func(w http.ResponseWriter, r *http.Request) { DeleteStudentClassCodeHandler(w, r, queries) }},
		{"POST invalid parameter", http.MethodPost, "/", url.Values{"student_id": {"nope"}, "class_code_id": {"10"}}, func(w http.ResponseWriter, r *http.Request) { DeleteStudentClassCodeHandler(w, r, queries) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := serveAuthenticatedStudentClassRequest(t, test.method, test.target, test.values, test.handle)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d, want 400", response.Code)
			}
		})
	}
}

func TestAddStudentClassCodeClassifiesDuplicateRelation(t *testing.T) {
	conn, queries := newStudentClassHandlerTestDB(t)
	response := serveAuthenticatedStudentClassRequest(t, http.MethodPost, "/", url.Values{"student_id": {"1"}, "class_code_id": {"10"}}, func(w http.ResponseWriter, r *http.Request) {
		AddStudentClassCodeHandler(w, r, queries)
	})
	assertStudentClassBusinessMessage(t, response, "Cette classe est déjà associée à cet élève.")
	assertStudentClassRelationCount(t, conn, 1, 10, 1)
}

func TestAddStudentClassCodeKeepsMissingAndForeignContextsAsNotFound(t *testing.T) {
	tests := []url.Values{
		{"student_id": {"999"}, "class_code_id": {"10"}},
		{"student_id": {"3"}, "class_code_id": {"10"}},
		{"student_id": {"1"}, "class_code_id": {"999"}},
		{"student_id": {"1"}, "class_code_id": {"20"}},
	}
	for _, values := range tests {
		_, queries := newStudentClassHandlerTestDB(t)
		response := serveAuthenticatedStudentClassRequest(t, http.MethodPost, "/", values, func(w http.ResponseWriter, r *http.Request) {
			AddStudentClassCodeHandler(w, r, queries)
		})
		if response.Code != http.StatusNotFound {
			t.Fatalf("values=%v status=%d, want 404", values, response.Code)
		}
	}
}

func TestAddStudentClassCodeReturnsInternalServerErrorForUnexpectedDBFailure(t *testing.T) {
	conn, queries := newStudentClassHandlerTestDB(t)
	if _, err := conn.Exec(`
		CREATE TRIGGER fail_student_class_insert BEFORE INSERT ON student_class_codes
		BEGIN SELECT RAISE(ABORT, 'forced unexpected relation failure'); END;
	`); err != nil {
		t.Fatal(err)
	}
	response := serveAuthenticatedStudentClassRequest(t, http.MethodPost, "/", url.Values{"student_id": {"1"}, "class_code_id": {"11"}}, func(w http.ResponseWriter, r *http.Request) {
		AddStudentClassCodeHandler(w, r, queries)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", response.Code)
	}
	assertStudentClassRelationCount(t, conn, 1, 11, 0)
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

func serveAuthenticatedStudentClassRequest(t *testing.T, method, target string, values url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "student-class-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	var body *strings.Reader
	if values == nil {
		body = strings.NewReader("")
	} else {
		body = strings.NewReader(values.Encode())
	}
	request := httptest.NewRequest(method, target, body)
	if values != nil {
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

func withStudentClassRepositoryRoot(t *testing.T, run func()) {
	t.Helper()
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../../.."); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(current) }()
	run()
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

func assertStudentClassBusinessMessage(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want 303", response.Code)
	}
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Path != data.ErrorMessageURL || location.Query().Get("errormessage") != want {
		t.Fatalf("location=%q message=%q", response.Header().Get("Location"), location.Query().Get("errormessage"))
	}
}
