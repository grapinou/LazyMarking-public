package qcm

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

func TestQCMHandlersReturnNotFoundForMissingOrForeignQCM(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`CREATE TABLE qcm (id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL); INSERT INTO qcm VALUES (1, 'owned', 1), (2, 'foreign', 2);`); err != nil {
		t.Fatal(err)
	}
	queries := db.New(conn)
	tests := []struct {
		name   string
		method string
		target string
		form   url.Values
		serve  http.HandlerFunc
	}{
		{name: "update missing", method: http.MethodPost, target: "/", form: url.Values{"qcm_id": {"999"}, "new_qcm": {"new"}}, serve: func(w http.ResponseWriter, r *http.Request) { EditQCMHandler(w, r, queries) }},
		{name: "delete foreign", method: http.MethodPost, target: "/", form: url.Values{"qcm_id": {"2"}}, serve: func(w http.ResponseWriter, r *http.Request) { DeleteQCMHandler(w, r, queries) }},
		{name: "edit form foreign", method: http.MethodGet, target: "/?qcm_id=2", serve: func(w http.ResponseWriter, r *http.Request) { EditFormQCMHandler(w, r, queries) }},
		{name: "delete form missing", method: http.MethodGet, target: "/?qcm_id=999", serve: func(w http.ResponseWriter, r *http.Request) { DeleteFormQCMHandler(w, r, queries) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedQCMRequest(t, tc.method, tc.target, tc.form, tc.serve)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
		})
	}
	var name string
	if err := conn.QueryRow("SELECT name FROM qcm WHERE id = 2 AND user_id = 2").Scan(&name); err != nil || name != "foreign" {
		t.Fatalf("foreign QCM name = %q, err = %v", name, err)
	}
}

func serveAuthenticatedQCMRequest(t *testing.T, method, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "qcm-handler-test-key-32-bytes-long")
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
