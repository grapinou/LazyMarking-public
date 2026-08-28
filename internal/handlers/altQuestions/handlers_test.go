package altquestions

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

func authenticatedAltQuestionPost(t *testing.T, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "alt-question-handler-test-key-32-bytes")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
