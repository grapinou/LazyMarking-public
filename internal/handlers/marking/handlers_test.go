package marking

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

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
