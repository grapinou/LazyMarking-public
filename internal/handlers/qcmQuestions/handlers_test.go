package qcmquestions

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

func TestQCMQuestionManagementPagesRequireOwnedQCM(t *testing.T) {
	conn, queries := newQCMQuestionHandlerTestDB(t)
	_ = conn
	tests := []struct {
		name   string
		target string
		serve  http.HandlerFunc
	}{
		{name: "table missing", target: "/?qcm_id=999", serve: func(w http.ResponseWriter, r *http.Request) { TableQCMQuestionsHandler(w, r, queries) }},
		{name: "table foreign", target: "/?qcm_id=2", serve: func(w http.ResponseWriter, r *http.Request) { TableQCMQuestionsHandler(w, r, queries) }},
		{name: "add form missing", target: "/?qcm_id=999", serve: func(w http.ResponseWriter, r *http.Request) { AddFormQCMQuestionHandler(w, r, queries) }},
		{name: "add form foreign", target: "/?qcm_id=2", serve: func(w http.ResponseWriter, r *http.Request) { AddFormQCMQuestionHandler(w, r, queries) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedQCMQuestionRequest(t, http.MethodGet, tc.target, nil, tc.serve)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
		})
	}
}

func TestAddQCMQuestionsRollsBackMixedOwnershipSelection(t *testing.T) {
	conn, queries := newQCMQuestionHandlerTestDB(t)
	form := url.Values{"qcm_id": {"3"}, "question_ids": {"10", "20"}}
	response := serveAuthenticatedQCMQuestionRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
		AddQCMQuestionHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM qcm_questions WHERE qcm_id = 3").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("partially inserted QCM relations = %d, want 0", count)
	}
}

func TestDeleteQCMQuestionDoesNotReportMissingOrForeignAsSuccess(t *testing.T) {
	conn, queries := newQCMQuestionHandlerTestDB(t)
	tests := []struct {
		name string
		form url.Values
	}{
		{name: "missing relation", form: url.Values{"qcm_id": {"1"}, "qcm_question_id": {"999"}}},
		{name: "foreign relation", form: url.Values{"qcm_id": {"2"}, "qcm_question_id": {"200"}}},
		{name: "mismatched parent", form: url.Values{"qcm_id": {"3"}, "qcm_question_id": {"100"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedQCMQuestionRequest(t, http.MethodPost, "/", tc.form, func(w http.ResponseWriter, r *http.Request) {
				DeleteQCMQuestionHandler(w, r, queries)
			})
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
		})
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM qcm_questions WHERE id = 200 AND user_id = 2").Scan(&count); err != nil || count != 1 {
		t.Fatalf("foreign relation count = %d, err = %v; want 1", count, err)
	}
}

func newQCMQuestionHandlerTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE qcm (id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE questions (id INTEGER PRIMARY KEY, content TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE qcm_questions (id INTEGER PRIMARY KEY, qcm_id INTEGER NOT NULL, question_id INTEGER NOT NULL, user_id INTEGER NOT NULL, UNIQUE(qcm_id, question_id));
		INSERT INTO qcm VALUES (1, 'owned', 1), (2, 'foreign', 2), (3, 'owned empty', 1);
		INSERT INTO questions VALUES (10, 'owned content', 1), (20, 'foreign content', 2);
		INSERT INTO qcm_questions VALUES (100, 1, 10, 1), (200, 2, 20, 2);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedQCMQuestionRequest(t *testing.T, method, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "qcm-question-handler-test-key-long")
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
