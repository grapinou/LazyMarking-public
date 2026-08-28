package questions

import (
	"context"
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

func TestLoadQuestionFamiliesBuildsOwnedFamiliesIncludingEmptyFamily(t *testing.T) {
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
INSERT INTO subjects VALUES(1,1),(2,2); INSERT INTO themes VALUES(1,1),(2,2); INSERT INTO year_levels VALUES(1,1),(2,2);
INSERT INTO skills VALUES(1,1),(2,2); INSERT INTO difficulties VALUES(1,1),(2,2); INSERT INTO points VALUES(1,1),(2,2);
INSERT INTO questions VALUES(10,1,1,1,1,1,1,'owned with variants',1),(11,1,1,1,1,1,1,'owned empty',1),(20,2,2,2,2,2,2,'foreign main',2);
INSERT INTO alt_questions VALUES(100,10,'owned variant A',1),(101,10,'owned variant B',1),(102,10,'foreign variant',2),(200,20,'foreign parent variant',1);`); err != nil {
		t.Fatal(err)
	}

	families, err := loadQuestionFamilies(context.Background(), db.New(conn), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(families) != 2 {
		t.Fatalf("families count = %d, want 2", len(families))
	}
	if families[0].Main.Content != "owned empty" || families[0].Variants == nil || len(families[0].Variants) != 0 {
		t.Fatalf("first family = %#v, want highest-ID empty family first", families[0])
	}
	if families[1].Main.Content != "owned with variants" || len(families[1].Variants) != 2 {
		t.Fatalf("second family = %#v, want family with two variants", families[1])
	}
	for _, family := range families {
		if family.Main.Content == "foreign main" {
			t.Fatal("foreign main question exposed")
		}
		for _, variant := range family.Variants {
			if variant.Content == "foreign variant" || variant.Content == "foreign parent variant" {
				t.Fatalf("foreign variant exposed: %#v", variant)
			}
		}
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
