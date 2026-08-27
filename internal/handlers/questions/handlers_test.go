package questions

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
)

func TestQuestionFormsReturnNotFoundForMissingOrForeignQuestion(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	_, err = conn.Exec(`
CREATE TABLE subjects(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE themes(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE year_levels(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE skills(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE difficulties(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE points(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE questions(id INTEGER PRIMARY KEY,subject_id INTEGER,theme_id INTEGER,year_level_id INTEGER,skill_id INTEGER,difficulty_id INTEGER,point_id INTEGER,content TEXT,user_id INTEGER);
INSERT INTO subjects VALUES(1,1),(2,2); INSERT INTO themes VALUES(1,1),(2,2); INSERT INTO year_levels VALUES(1,1),(2,2);
INSERT INTO skills VALUES(1,1),(2,2); INSERT INTO difficulties VALUES(1,1),(2,2); INSERT INTO points VALUES(1,1),(2,2);
INSERT INTO questions VALUES(1,1,1,1,1,1,1,'owned',1),(2,2,2,2,2,2,2,'foreign',2);`)
	if err != nil {
		t.Fatal(err)
	}
	queries := db.New(conn)
	for _, tc := range []struct {
		name   string
		target string
		serve  http.HandlerFunc
	}{
		{"missing edit", "/?question_id=999", func(w http.ResponseWriter, r *http.Request) { EditFormQuestionHandler(w, r, queries) }},
		{"foreign edit", "/?question_id=2", func(w http.ResponseWriter, r *http.Request) { EditFormQuestionHandler(w, r, queries) }},
		{"missing delete", "/?question_id=999", func(w http.ResponseWriter, r *http.Request) { DeleteFormQuestionHandler(w, r, queries) }},
		{"foreign delete", "/?question_id=2", func(w http.ResponseWriter, r *http.Request) { DeleteFormQuestionHandler(w, r, queries) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := authenticatedQuestionRequest(t, tc.target, tc.serve)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want 404", response.Code)
			}
		})
	}
}

func authenticatedQuestionRequest(t *testing.T, target string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "question-handler-test-key-32-bytes")
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
