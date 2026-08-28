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

func TestTableAltAnswersHandlerRejectsMissingForeignAndMismatchedParents(t *testing.T) {
	queries := db.New(setupAltAnswerTableHandlerTest(t))
	tests := []struct {
		name   string
		target string
	}{
		{
			name:   "missing question",
			target: "/dashboard/questions/altquestions/altanswers?question_id=999&alt_question_id=10",
		},
		{
			name:   "foreign question and variant",
			target: "/dashboard/questions/altquestions/altanswers?question_id=2&alt_question_id=20",
		},
		{
			name:   "missing variant",
			target: "/dashboard/questions/altquestions/altanswers?question_id=1&alt_question_id=999",
		},
		{
			name:   "foreign variant under owned question parameter",
			target: "/dashboard/questions/altquestions/altanswers?question_id=1&alt_question_id=20",
		},
		{
			name:   "owned variant bound to another owned question",
			target: "/dashboard/questions/altquestions/altanswers?question_id=1&alt_question_id=11",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := authenticatedAltAnswerRequest(t, http.MethodGet, test.target, nil, func(w http.ResponseWriter, r *http.Request) {
				TableAltAnswersHandler(w, r, queries)
			})
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want %d", response.Code, http.StatusNotFound)
			}
		})
	}
}

func TestAltAnswerFormsRejectForeignAndMismatchedParents(t *testing.T) {
	queries := db.New(setupAltAnswerTableHandlerTest(t))
	tests := []struct {
		name    string
		target  string
		handler http.HandlerFunc
	}{
		{
			name:   "add with missing question",
			target: "/dashboard/questions/altquestions/altanswers/add?question_id=999&alt_question_id=10",
			handler: func(w http.ResponseWriter, r *http.Request) {
				AddFormAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "add with foreign question and variant",
			target: "/dashboard/questions/altquestions/altanswers/add?question_id=2&alt_question_id=20",
			handler: func(w http.ResponseWriter, r *http.Request) {
				AddFormAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "add with missing variant",
			target: "/dashboard/questions/altquestions/altanswers/add?question_id=1&alt_question_id=999",
			handler: func(w http.ResponseWriter, r *http.Request) {
				AddFormAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "add with foreign variant",
			target: "/dashboard/questions/altquestions/altanswers/add?question_id=1&alt_question_id=20",
			handler: func(w http.ResponseWriter, r *http.Request) {
				AddFormAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "add with variant from another owned question",
			target: "/dashboard/questions/altquestions/altanswers/add?question_id=1&alt_question_id=11",
			handler: func(w http.ResponseWriter, r *http.Request) {
				AddFormAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "edit with missing question",
			target: "/dashboard/questions/altquestions/altanswers/edit?question_id=999&alt_question_id=10&alt_answer_id=100",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "edit with foreign chain",
			target: "/dashboard/questions/altquestions/altanswers/edit?question_id=2&alt_question_id=20&alt_answer_id=200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "edit with missing variant",
			target: "/dashboard/questions/altquestions/altanswers/edit?question_id=1&alt_question_id=999&alt_answer_id=100",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "edit with foreign variant",
			target: "/dashboard/questions/altquestions/altanswers/edit?question_id=1&alt_question_id=20&alt_answer_id=200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "edit with missing answer",
			target: "/dashboard/questions/altquestions/altanswers/edit?question_id=1&alt_question_id=10&alt_answer_id=999",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "edit with foreign answer",
			target: "/dashboard/questions/altquestions/altanswers/edit?question_id=1&alt_question_id=10&alt_answer_id=200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "edit with answer from another variant",
			target: "/dashboard/questions/altquestions/altanswers/edit?question_id=1&alt_question_id=10&alt_answer_id=110",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAltAnswerHandler(w, r, queries)
			},
		},
		{
			name:   "edit with variant from another owned question",
			target: "/dashboard/questions/altquestions/altanswers/edit?question_id=1&alt_question_id=11&alt_answer_id=110",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAltAnswerHandler(w, r, queries)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := authenticatedAltAnswerRequest(t, http.MethodGet, test.target, nil, test.handler)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want %d", response.Code, http.StatusNotFound)
			}
		})
	}
}

func setupAltAnswerTableHandlerTest(t *testing.T) *sql.DB {
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
CREATE TABLE alt_questions(id INTEGER PRIMARY KEY,question_id INTEGER,content TEXT,user_id INTEGER);
CREATE TABLE alt_answers(id INTEGER PRIMARY KEY,alt_question_id INTEGER,state INTEGER,content TEXT,user_id INTEGER);
INSERT INTO subjects VALUES(1,1),(2,2); INSERT INTO themes VALUES(1,1),(2,2); INSERT INTO year_levels VALUES(1,1),(2,2);
INSERT INTO skills VALUES(1,1),(2,2); INSERT INTO difficulties VALUES(1,1),(2,2); INSERT INTO points VALUES(1,1),(2,2);
INSERT INTO questions VALUES(1,1,1,1,1,1,1,'owned question',1),(2,2,2,2,2,2,2,'foreign question',2),(3,1,1,1,1,1,1,'other owned question',1);
INSERT INTO alt_questions VALUES(10,1,'owned variant',1),(11,3,'other owned variant',1),(20,2,'foreign variant',2);
INSERT INTO alt_answers VALUES(100,10,1,'owned answer',1),(110,11,0,'other owned answer',1),(200,20,1,'foreign answer',2);`); err != nil {
		t.Fatal(err)
	}
	return conn
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
