package answers

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

func TestAddAnswerHandlerPreservesSpecialCharacters(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(
		"CREATE TABLE questions(id INTEGER PRIMARY KEY, user_id INTEGER);" +
			"CREATE TABLE answers(id INTEGER PRIMARY KEY, question_id INTEGER, state INTEGER, content TEXT, user_id INTEGER);" +
			"INSERT INTO questions VALUES(1, 1);",
	); err != nil {
		t.Fatal(err)
	}

	want := "On appelle cela une réponse \"correcte\".\\µ ° é"
	response := authenticatedAnswerPost(t, "/dashboard/questions/answers/add", url.Values{
		"question_id": {"1"},
		"state":       {"1"},
		"content":     {want},
	}, func(w http.ResponseWriter, r *http.Request) {
		AddAnswerHandler(w, r, db.New(conn))
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusSeeOther)
	}

	var got string
	if err := conn.QueryRow("SELECT content FROM answers").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("stored content = %q, want unchanged content %q", got, want)
	}
}

func TestAnswerFormsRejectMissingForeignAndMismatchedParents(t *testing.T) {
	conn := setupAnswerFormHandlerTest(t)
	queries := db.New(conn)
	tests := []struct {
		name    string
		target  string
		handler http.HandlerFunc
	}{
		{
			name:   "add missing question",
			target: "/dashboard/questions/answers/add?question_id=999",
			handler: func(w http.ResponseWriter, r *http.Request) {
				AddFormAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "add foreign question",
			target: "/dashboard/questions/answers/add?question_id=2",
			handler: func(w http.ResponseWriter, r *http.Request) {
				AddFormAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "edit missing answer",
			target: "/dashboard/questions/answers/edit?question_id=1&answer_id=999",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "edit foreign answer and question",
			target: "/dashboard/questions/answers/edit?question_id=2&answer_id=20",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "edit answer bound to another owned question",
			target: "/dashboard/questions/answers/edit?question_id=1&answer_id=11",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAnswerHandler(w, r, queries)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := authenticatedAnswerRequest(t, http.MethodGet, test.target, nil, test.handler)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want %d", response.Code, http.StatusNotFound)
			}
		})
	}
}

func setupAnswerFormHandlerTest(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`
CREATE TABLE subjects(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE themes(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE year_levels(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE skills(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE difficulties(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE points(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE questions(id INTEGER PRIMARY KEY,subject_id INTEGER,theme_id INTEGER,year_level_id INTEGER,skill_id INTEGER,difficulty_id INTEGER,point_id INTEGER,content TEXT,user_id INTEGER);
CREATE TABLE answers(id INTEGER PRIMARY KEY,question_id INTEGER,state INTEGER,content TEXT,user_id INTEGER);
INSERT INTO subjects VALUES(1,1),(2,2); INSERT INTO themes VALUES(1,1),(2,2); INSERT INTO year_levels VALUES(1,1),(2,2);
INSERT INTO skills VALUES(1,1),(2,2); INSERT INTO difficulties VALUES(1,1),(2,2); INSERT INTO points VALUES(1,1),(2,2);
INSERT INTO questions VALUES(1,1,1,1,1,1,1,'owned question',1),(2,2,2,2,2,2,2,'foreign question',2),(3,1,1,1,1,1,1,'other owned question',1);
INSERT INTO answers VALUES(10,1,1,'owned answer',1),(11,3,0,'other owned answer',1),(20,2,1,'foreign answer',2);`); err != nil {
		t.Fatal(err)
	}
	return conn
}

func authenticatedAnswerPost(t *testing.T, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	return authenticatedAnswerRequest(t, http.MethodPost, target, form, handler)
}

func authenticatedAnswerRequest(t *testing.T, method, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "answer-handler-test-key-32-bytes")
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
