package altanswers

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

func TestAddAltAnswerHandlerPreservesSpecialCharacters(t *testing.T) {
	conn := setupAltAnswerHandlerTest(t)
	want := "La valeur est \"1,0 kg/L\".\\µ ° é"
	response := authenticatedAltAnswerRequest(t, http.MethodPost, "/dashboard/questions/altquestions/altanswers/add", url.Values{
		"question_id":     {"1"},
		"alt_question_id": {"2"},
		"state":           {"1"},
		"content":         {want},
	}, func(w http.ResponseWriter, r *http.Request) {
		AddAltAnswerHandler(w, r, db.New(conn))
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusSeeOther)
	}

	var got string
	if err := conn.QueryRow("SELECT content FROM alt_answers").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("stored content = %q, want unchanged content %q", got, want)
	}
}

func TestAltAnswerHandlersRejectMissingAltQuestionID(t *testing.T) {
	queries := db.New(setupAltAnswerHandlerTest(t))
	tests := []struct {
		name    string
		method  string
		target  string
		form    url.Values
		handler http.HandlerFunc
	}{
		{
			name:   "add",
			method: http.MethodPost,
			target: "/dashboard/questions/altquestions/altanswers/add",
			form:   url.Values{"question_id": {"1"}},
			handler: func(w http.ResponseWriter, r *http.Request) {
				AddAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "edit",
			method: http.MethodPost,
			target: "/dashboard/questions/altquestions/altanswers/edit",
			form:   url.Values{"question_id": {"1"}},
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "delete form",
			method: http.MethodGet,
			target: "/dashboard/questions/altquestions/altanswers/delete?question_id=1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				DeleteFormAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "delete",
			method: http.MethodPost,
			target: "/dashboard/questions/altquestions/altanswers/delete",
			form:   url.Values{"question_id": {"1"}},
			handler: func(w http.ResponseWriter, r *http.Request) {
				DeleteAltAnswerHandler(w, r, queries)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := authenticatedAltAnswerRequest(t, test.method, test.target, test.form, test.handler)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d, want %d", response.Code, http.StatusBadRequest)
			}
		})
	}
}

func setupAltAnswerHandlerTest(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(
		"CREATE TABLE alt_questions(id INTEGER PRIMARY KEY, question_id INTEGER, user_id INTEGER);" +
			"CREATE TABLE alt_answers(id INTEGER PRIMARY KEY, alt_question_id INTEGER, state INTEGER, content TEXT, user_id INTEGER);" +
			"INSERT INTO alt_questions VALUES(2, 1, 1);",
	); err != nil {
		t.Fatal(err)
	}
	return conn
}

func authenticatedAltAnswerRequest(t *testing.T, method, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "alt-answer-handler-test-key-32-bytes")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	var body string
	if form != nil {
		body = form.Encode()
	}
	request := httptest.NewRequest(method, target, strings.NewReader(body))
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
