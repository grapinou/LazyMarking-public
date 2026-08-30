package classcodes

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestAddClassCodeClassifiesUniqueAndCheckConstraints(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		message string
	}{
		{"duplicate", "Owned", "Cette classe existe déjà."},
		{"blank", "   ", "Le nom de la classe doit être renseigné."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn, queries := newClassCodeHandlerTestDB(t)
			response := serveAuthenticatedClassCodeRequest(t, url.Values{"class_code": {test.value}}, func(w http.ResponseWriter, r *http.Request) {
				AddClassCodeHandler(w, r, queries)
			})
			assertClassCodeBusinessMessage(t, response, test.message)
			assertClassCodeCount(t, conn, 1, 2)
		})
	}
}

func TestAddClassCodeReturnsInternalServerErrorForUnexpectedDBFailure(t *testing.T) {
	conn, queries := newClassCodeHandlerTestDB(t)
	if _, err := conn.Exec(`
		CREATE TRIGGER fail_class_insert BEFORE INSERT ON class_codes
		BEGIN SELECT RAISE(ABORT, 'forced unexpected insert failure'); END;
	`); err != nil {
		t.Fatal(err)
	}
	response := serveAuthenticatedClassCodeRequest(t, url.Values{"class_code": {"New class"}}, func(w http.ResponseWriter, r *http.Request) {
		AddClassCodeHandler(w, r, queries)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", response.Code)
	}
	assertClassCodeCount(t, conn, 1, 2)
}

func TestAddClassCodePreservesDoubleQuotes(t *testing.T) {
	conn, queries := newClassCodeHandlerTestDB(t)
	want := `6e "A"`
	response := serveAuthenticatedClassCodeRequest(t, url.Values{"class_code": {"  " + want + "  "}}, func(w http.ResponseWriter, r *http.Request) {
		AddClassCodeHandler(w, r, queries)
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != data.DefaultStudentRoutes.ClassCodesURL {
		t.Fatalf("status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
	var stored string
	if err := conn.QueryRow("SELECT name FROM class_codes WHERE user_id = 1 AND name = ?", want).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != want {
		t.Fatalf("stored name=%q, want %q", stored, want)
	}
}

func TestEditClassCodeClassifiesUniqueAndCheckConstraints(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		message string
	}{
		{"duplicate", "Owned", "Cette classe existe déjà."},
		{"blank", "   ", "Le nom de la classe doit être renseigné."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn, queries := newClassCodeHandlerTestDB(t)
			form := url.Values{"class_code_id": {"11"}, "new_class_code": {test.value}}
			response := serveAuthenticatedClassCodeRequest(t, form, func(w http.ResponseWriter, r *http.Request) {
				EditClassCodeHandler(w, r, queries)
			})
			assertClassCodeBusinessMessage(t, response, test.message)
			assertClassCodeNamed(t, conn, 11, "Other")
		})
	}
}

func TestEditClassCodeReturnsInternalServerErrorForUnexpectedDBFailure(t *testing.T) {
	conn, queries := newClassCodeHandlerTestDB(t)
	if _, err := conn.Exec(`
		CREATE TRIGGER fail_class_update BEFORE UPDATE ON class_codes
		BEGIN SELECT RAISE(ABORT, 'forced unexpected update failure'); END;
	`); err != nil {
		t.Fatal(err)
	}
	form := url.Values{"class_code_id": {"11"}, "new_class_code": {"Changed"}}
	response := serveAuthenticatedClassCodeRequest(t, form, func(w http.ResponseWriter, r *http.Request) {
		EditClassCodeHandler(w, r, queries)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", response.Code)
	}
	assertClassCodeNamed(t, conn, 11, "Other")
}

func TestEditClassCodePreservesDoubleQuotes(t *testing.T) {
	conn, queries := newClassCodeHandlerTestDB(t)
	want := `6e "Alpha"`
	form := url.Values{"class_code_id": {"11"}, "new_class_code": {"  " + want + "  "}}
	response := serveAuthenticatedClassCodeRequest(t, form, func(w http.ResponseWriter, r *http.Request) {
		EditClassCodeHandler(w, r, queries)
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != data.DefaultStudentRoutes.ClassCodesURL {
		t.Fatalf("status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
	assertClassCodeNamed(t, conn, 11, want)
}

func TestEditClassCodeKeepsOwnedZeroRowContract(t *testing.T) {
	_, queries := newClassCodeHandlerTestDB(t)
	for _, id := range []string{"20", "999"} {
		response := serveAuthenticatedClassCodeRequest(t, url.Values{"class_code_id": {id}, "new_class_code": {"Forged"}}, func(w http.ResponseWriter, r *http.Request) {
			EditClassCodeHandler(w, r, queries)
		})
		if response.Code != http.StatusNotFound {
			t.Fatalf("id=%s status=%d, want 404", id, response.Code)
		}
	}
}

func newClassCodeHandlerTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE class_codes (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL CHECK(length(trim(name)) > 0),
			user_id INTEGER NOT NULL,
			UNIQUE(name, user_id)
		);
		INSERT INTO class_codes VALUES (10, 'Owned', 1), (11, 'Other', 1), (20, 'Foreign', 2);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedClassCodeRequest(t *testing.T, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "class-code-handler-test-key-32-bytes")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

func assertClassCodeBusinessMessage(t *testing.T, response *httptest.ResponseRecorder, want string) {
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

func assertClassCodeCount(t *testing.T, conn *sql.DB, userID int64, want int) {
	t.Helper()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM class_codes WHERE user_id = ?", userID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("class count=%d, want %d", count, want)
	}
}

func assertClassCodeNamed(t *testing.T, conn *sql.DB, id int64, want string) {
	t.Helper()
	var name string
	if err := conn.QueryRow("SELECT name FROM class_codes WHERE id = ?", id).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != want {
		t.Fatalf("class %d name=%q, want %q", id, name, want)
	}
}
