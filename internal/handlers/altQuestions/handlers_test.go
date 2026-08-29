package altquestions

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
)

func TestDeleteAltQuestionHandlerRemovesIllustratedVariantAndFile(t *testing.T) {
	t.Chdir(t.TempDir())
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`
CREATE TABLE questions(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE alt_questions(id INTEGER PRIMARY KEY,question_id INTEGER,content TEXT,user_id INTEGER);
CREATE TABLE alt_images(id INTEGER PRIMARY KEY,alt_question_id INTEGER,image_name TEXT,resize_percentage INTEGER,user_id INTEGER);
INSERT INTO questions VALUES(42,1);
INSERT INTO alt_questions VALUES(7,42,'illustrated variant',1);
INSERT INTO alt_images VALUES(70,7,'variant-7.png',50,1);`); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(config.ImageSavePath, 0o750); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(config.ImageSavePath, "variant-7.png")
	if err := os.WriteFile(imagePath, []byte("image"), 0o640); err != nil {
		t.Fatal(err)
	}

	response := authenticatedAltQuestionPost(t, "/delete", url.Values{
		"question_id":     {"42"},
		"alt_question_id": {"7"},
	}, func(w http.ResponseWriter, r *http.Request) { DeleteAltQuestionHandler(w, r, db.New(conn)) })
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusSeeOther)
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM alt_questions WHERE id=7").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("remaining variants=%d, want 0", count)
	}
	if _, err := os.Stat(imagePath); !os.IsNotExist(err) {
		t.Fatalf("stored variant image was not removed: %v", err)
	}
}

func TestAddAltQuestionHandlerPreservesSpecialCharacters(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(
		"CREATE TABLE questions(id INTEGER PRIMARY KEY, user_id INTEGER);" +
			"CREATE TABLE alt_questions(id INTEGER PRIMARY KEY, question_id INTEGER, content TEXT, user_id INTEGER);" +
			"INSERT INTO questions VALUES(1, 1);",
	); err != nil {
		t.Fatal(err)
	}

	want := "Comment définir la \"masse volumique\" ?\\µ ° é"
	response := authenticatedAltQuestionPost(t, "/dashboard/questions/altquestions/add", url.Values{
		"question_id": {"1"},
		"content":     {want},
	}, func(w http.ResponseWriter, r *http.Request) {
		AddAltQuestionHandler(w, r, db.New(conn))
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusSeeOther)
	}

	var got string
	if err := conn.QueryRow("SELECT content FROM alt_questions").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("stored content = %q, want unchanged content %q", got, want)
	}
}

func TestAltQuestionFormsRejectMissingForeignAndMismatchedParents(t *testing.T) {
	conn := setupAltQuestionFormHandlerTest(t)
	queries := db.New(conn)
	tests := []struct {
		name    string
		target  string
		handler http.HandlerFunc
	}{
		{
			name:   "add missing question",
			target: "/dashboard/questions/altquestions/add?question_id=999",
			handler: func(w http.ResponseWriter, r *http.Request) {
				AddFormAltQuestionHandler(w, r, queries)
			},
		},
		{
			name:   "add foreign question",
			target: "/dashboard/questions/altquestions/add?question_id=2",
			handler: func(w http.ResponseWriter, r *http.Request) {
				AddFormAltQuestionHandler(w, r, queries)
			},
		},
		{
			name:   "edit missing variant",
			target: "/dashboard/questions/altquestions/edit?question_id=1&alt_question_id=999",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAltQuestionHandler(w, r, queries)
			},
		},
		{
			name:   "edit foreign variant",
			target: "/dashboard/questions/altquestions/edit?question_id=2&alt_question_id=20",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAltQuestionHandler(w, r, queries)
			},
		},
		{
			name:   "edit variant bound to another owned question",
			target: "/dashboard/questions/altquestions/edit?question_id=1&alt_question_id=11",
			handler: func(w http.ResponseWriter, r *http.Request) {
				EditFormAltQuestionHandler(w, r, queries)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := authenticatedAltQuestionRequest(t, http.MethodGet, test.target, nil, test.handler)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want %d", response.Code, http.StatusNotFound)
			}
		})
	}
}

func setupAltQuestionFormHandlerTest(t *testing.T) *sql.DB {
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
INSERT INTO subjects VALUES(1,1),(2,2); INSERT INTO themes VALUES(1,1),(2,2); INSERT INTO year_levels VALUES(1,1),(2,2);
INSERT INTO skills VALUES(1,1),(2,2); INSERT INTO difficulties VALUES(1,1),(2,2); INSERT INTO points VALUES(1,1),(2,2);
INSERT INTO questions VALUES(1,1,1,1,1,1,1,'owned question',1),(2,2,2,2,2,2,2,'foreign question',2),(3,1,1,1,1,1,1,'other owned question',1);
INSERT INTO alt_questions VALUES(10,1,'owned variant',1),(11,3,'other owned variant',1),(20,2,'foreign variant',2);`); err != nil {
		t.Fatal(err)
	}
	return conn
}

func authenticatedAltQuestionPost(t *testing.T, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	return authenticatedAltQuestionRequest(t, http.MethodPost, target, form, handler)
}

func authenticatedAltQuestionRequest(t *testing.T, method, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "alt-question-handler-test-key-32-bytes")
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
