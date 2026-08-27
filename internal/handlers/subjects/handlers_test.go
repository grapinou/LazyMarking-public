package subjects

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

type subjectDBHandler func(http.ResponseWriter, *http.Request, *db.Queries)

func TestOwnedSubjectHandlersDoNotReportZeroRowsAsSuccess(t *testing.T) {
	conn, queries := newSubjectHandlerTestDB(t)
	tests := []struct {
		name    string
		handler subjectDBHandler
		form    url.Values
	}{
		{name: "update missing", handler: EditSubjectHandler, form: url.Values{"subject_id": {"999"}, "new_subject": {"changed"}}},
		{name: "update foreign owned", handler: EditSubjectHandler, form: url.Values{"subject_id": {"2"}, "new_subject": {"changed"}}},
		{name: "delete missing", handler: DeleteSubjectHandler, form: url.Values{"subject_id": {"999"}}},
		{name: "delete foreign owned", handler: DeleteSubjectHandler, form: url.Values{"subject_id": {"2"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedSubjectRequest(t, http.MethodPost, "/", tc.form, tc.handler, queries)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
		})
	}

	var foreignName string
	if err := conn.QueryRow("SELECT name FROM subjects WHERE id = 2 AND user_id = 2").Scan(&foreignName); err != nil || foreignName != "foreign" {
		t.Fatalf("foreign-owned subject = %q, err = %v; want unchanged", foreignName, err)
	}
}

func TestOwnedSubjectFormsReturnNotFound(t *testing.T) {
	_, queries := newSubjectHandlerTestDB(t)
	tests := []struct {
		name    string
		handler subjectDBHandler
		query   string
	}{
		{name: "edit missing", handler: EditFormSubjectHandler, query: "subject_id=999"},
		{name: "edit foreign owned", handler: EditFormSubjectHandler, query: "subject_id=2"},
		{name: "delete missing", handler: DeleteFormSubjectHandler, query: "subject_id=999"},
		{name: "delete foreign owned", handler: DeleteFormSubjectHandler, query: "subject_id=2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedSubjectRequest(t, http.MethodGet, "/?"+tc.query, nil, tc.handler, queries)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
		})
	}
}

func newSubjectHandlerTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE subjects (id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL, UNIQUE(name, user_id));
		INSERT INTO subjects (id, name, user_id) VALUES (1, 'owned', 1), (2, 'foreign', 2);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedSubjectRequest(t *testing.T, method, target string, form url.Values, handler subjectDBHandler, queries *db.Queries) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "reference-handler-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}

	var body *strings.Reader
	if form == nil {
		body = strings.NewReader("")
	} else {
		body = strings.NewReader(form.Encode())
	}
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
	cookieResponse := httptest.NewRecorder()
	if err := session.Save(request, cookieResponse); err != nil {
		t.Fatal(err)
	}
	for _, cookie := range cookieResponse.Result().Cookies() {
		request.AddCookie(cookie)
	}

	response := httptest.NewRecorder()
	login.CheckAuth(tools.HandlerWithDB(handler, queries)).ServeHTTP(response, request)
	return response
}
