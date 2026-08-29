package tools_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/difficulties"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/points"
	"github.com/grapinou/LazyMarking/internal/handlers/skills"
	"github.com/grapinou/LazyMarking/internal/handlers/subjects"
	"github.com/grapinou/LazyMarking/internal/handlers/themes"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/handlers/yearlevels"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

type referenceDeleteHandler func(http.ResponseWriter, *http.Request, *db.Queries)

func TestReferenceDeleteHandlersClassifyDatabaseErrors(t *testing.T) {
	tests := []struct {
		name       string
		table      string
		formKey    string
		questionFK string
		listURL    string
		handler    referenceDeleteHandler
	}{
		{name: "subject", table: "subjects", formKey: "subject_id", questionFK: "subject_id", listURL: data.DefaultQuestionRoutes.SubjectsURL, handler: subjects.DeleteSubjectHandler},
		{name: "theme", table: "themes", formKey: "theme_id", questionFK: "theme_id", listURL: data.DefaultQuestionRoutes.ThemesURL, handler: themes.DeleteThemeHandler},
		{name: "year level", table: "year_levels", formKey: "yearlevel_id", questionFK: "year_level_id", listURL: data.DefaultQuestionRoutes.YearLevelsURL, handler: yearlevels.DeleteYearLevelHandler},
		{name: "skill", table: "skills", formKey: "skill_id", questionFK: "skill_id", listURL: data.DefaultQuestionRoutes.SkillsURL, handler: skills.DeleteSkillHandler},
		{name: "difficulty", table: "difficulties", formKey: "difficulty_id", questionFK: "difficulty_id", listURL: data.DefaultQuestionRoutes.DifficultiesURL, handler: difficulties.DeleteDifficultyHandler},
		{name: "point", table: "points", formKey: "point_id", questionFK: "point_id", listURL: data.DefaultQuestionRoutes.PointsURL, handler: points.DeletePointHandler},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, queries := newReferenceDeleteHandlerDB(t)

			response := serveAuthenticatedReferenceDelete(t, url.Values{tc.formKey: {"1"}}, tc.handler, queries)
			assertReferenceDeleteResponse(t, response, http.StatusSeeOther, tc.listURL)
			assertReferenceRowCount(t, conn, tc.table, 1, 0)

			response = serveAuthenticatedReferenceDelete(t, url.Values{tc.formKey: {"999"}}, tc.handler, queries)
			assertReferenceDeleteResponse(t, response, http.StatusNotFound, "")

			response = serveAuthenticatedReferenceDelete(t, url.Values{tc.formKey: {"2"}}, tc.handler, queries)
			assertReferenceDeleteResponse(t, response, http.StatusNotFound, "")
			assertReferenceRowCount(t, conn, tc.table, 2, 1)

			response = serveAuthenticatedReferenceDelete(t, url.Values{tc.formKey: {"3"}}, tc.handler, queries)
			assertReferenceDeleteResponse(t, response, http.StatusSeeOther, data.ErrorMessageURL)
			assertReferenceRowCount(t, conn, tc.table, 3, 1)
			assertQuestionReference(t, conn, tc.questionFK, 3)

			if _, err := conn.Exec("CREATE TRIGGER forced_delete_failure BEFORE DELETE ON " + tc.table + " BEGIN SELECT missing_delete_function(); END;"); err != nil {
				t.Fatal(err)
			}
			response = serveAuthenticatedReferenceDelete(t, url.Values{tc.formKey: {"4"}}, tc.handler, queries)
			assertReferenceDeleteResponse(t, response, http.StatusInternalServerError, "")
			assertReferenceRowCount(t, conn, tc.table, 4, 1)
		})
	}
}

func newReferenceDeleteHandlerDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)

	if _, err := conn.Exec(`
		CREATE TABLE subjects(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE themes(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE year_levels(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE skills(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE difficulties(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE points(id INTEGER PRIMARY KEY, point_value INTEGER NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE questions(
			id INTEGER PRIMARY KEY,
			subject_id INTEGER NOT NULL REFERENCES subjects(id) ON DELETE RESTRICT,
			theme_id INTEGER NOT NULL REFERENCES themes(id) ON DELETE RESTRICT,
			year_level_id INTEGER NOT NULL REFERENCES year_levels(id) ON DELETE RESTRICT,
			skill_id INTEGER NOT NULL REFERENCES skills(id) ON DELETE RESTRICT,
			difficulty_id INTEGER NOT NULL REFERENCES difficulties(id) ON DELETE RESTRICT,
			point_id INTEGER NOT NULL REFERENCES points(id) ON DELETE RESTRICT
		);
		INSERT INTO subjects VALUES(1,'free',1),(2,'foreign',2),(3,'used',1),(4,'broken',1);
		INSERT INTO themes VALUES(1,'free',1),(2,'foreign',2),(3,'used',1),(4,'broken',1);
		INSERT INTO year_levels VALUES(1,'free',1),(2,'foreign',2),(3,'used',1),(4,'broken',1);
		INSERT INTO skills VALUES(1,'free',1),(2,'foreign',2),(3,'used',1),(4,'broken',1);
		INSERT INTO difficulties VALUES(1,'free',1),(2,'foreign',2),(3,'used',1),(4,'broken',1);
		INSERT INTO points VALUES(1,1,1),(2,2,2),(3,3,1),(4,4,1);
		INSERT INTO questions VALUES(10,3,3,3,3,3,3);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedReferenceDelete(t *testing.T, form url.Values, handler referenceDeleteHandler, queries *db.Queries) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "reference-delete-test-key-32-bytes")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	session, err := login.GetStore().Get(request, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["user_id"] = int64(1)
	session.Values["username"] = "owner"
	cookies := httptest.NewRecorder()
	if err := session.Save(request, cookies); err != nil {
		t.Fatal(err)
	}
	for _, cookie := range cookies.Result().Cookies() {
		request.AddCookie(cookie)
	}

	response := httptest.NewRecorder()
	login.CheckAuth(tools.HandlerWithDB(handler, queries)).ServeHTTP(response, request)
	return response
}

func assertReferenceDeleteResponse(t *testing.T, response *httptest.ResponseRecorder, wantStatus int, wantPath string) {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("status=%d, want %d; body=%q", response.Code, wantStatus, response.Body.String())
	}
	if wantPath == "" {
		if response.Header().Get("Location") != "" {
			t.Fatalf("unexpected redirect %q", response.Header().Get("Location"))
		}
		return
	}
	location, err := response.Result().Location()
	if err != nil {
		t.Fatal(err)
	}
	if location.Path != wantPath {
		t.Fatalf("redirect path=%q, want %q", location.Path, wantPath)
	}
}

func assertReferenceRowCount(t *testing.T, conn *sql.DB, table string, id int64, want int) {
	t.Helper()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE id=?", id).Scan(&count); err != nil || count != want {
		t.Fatalf("%s id=%d count=%d err=%v, want %d", table, id, count, err, want)
	}
}

func assertQuestionReference(t *testing.T, conn *sql.DB, column string, want int64) {
	t.Helper()
	var got int64
	if err := conn.QueryRow("SELECT " + column + " FROM questions WHERE id=10").Scan(&got); err != nil || got != want {
		t.Fatalf("question %s=%d err=%v, want %d", column, got, err, want)
	}
}
