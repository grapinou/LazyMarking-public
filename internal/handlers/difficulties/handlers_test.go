package difficulties

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

type difficultyDBHandler func(http.ResponseWriter, *http.Request, *db.Queries)

func TestDifficultyListViewData(t *testing.T) {
	conn, queries := newDifficultyViewTestDB(t)
	if _, err := conn.Exec("INSERT INTO difficulties(id,name,user_id) VALUES(3,'second',1)"); err != nil {
		t.Fatal(err)
	}
	var page data.DifficultyPageData
	previous := renderTableDifficultyPage
	renderTableDifficultyPage = func(_ http.ResponseWriter, got data.DifficultyPageData) { page = got }
	defer func() { renderTableDifficultyPage = previous }()

	response := serveAuthenticatedDifficultyRequest(t, http.MethodGet, "/", nil, TableDifficultiesHandler, queries)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", response.Code)
	}
	if len(page.DifficultyItems) != 2 {
		t.Fatalf("items=%#v, want two", page.DifficultyItems)
	}
	wantIDs := []int64{1, 3}
	wantNames := []string{"owned", "second"}
	for i, item := range page.DifficultyItems {
		if item.ID != wantIDs[i] || item.Name != wantNames[i] {
			t.Fatalf("item %d=%#v", i, item)
		}
		assertDifficultyURL(t, item.EditURL, data.DefaultDifficultyRoutes.EditURL, item.ID)
		assertDifficultyURL(t, item.DeleteURL, data.DefaultDifficultyRoutes.DeleteURL, item.ID)
	}
	if _, err := conn.Exec("DELETE FROM difficulties WHERE user_id=1"); err != nil {
		t.Fatal(err)
	}
	page = data.DifficultyPageData{}
	serveAuthenticatedDifficultyRequest(t, http.MethodGet, "/", nil, TableDifficultiesHandler, queries)
	if page.DifficultyItems == nil || len(page.DifficultyItems) != 0 {
		t.Fatalf("empty items=%#v, want non-nil empty slice", page.DifficultyItems)
	}
}

func TestDifficultyFormsProvideContextAndCancelURL(t *testing.T) {
	_, queries := newDifficultyViewTestDB(t)
	t.Run("add", func(t *testing.T) {
		var page data.DifficultyPageData
		previous := renderAddFormDifficultyPage
		renderAddFormDifficultyPage = func(_ http.ResponseWriter, got data.DifficultyPageData) { page = got }
		defer func() { renderAddFormDifficultyPage = previous }()
		serveAuthenticatedDifficultyRequest(t, http.MethodGet, "/", nil, AddFormDifficultyHandler, queries)
		if page.CancelURL != data.DefaultQuestionRoutes.DifficultiesURL {
			t.Fatalf("CancelURL=%q", page.CancelURL)
		}
	})
	for _, tc := range []struct {
		name    string
		handler difficultyDBHandler
		set     func(func(http.ResponseWriter, data.DifficultyPageData)) func()
	}{
		{name: "edit", handler: EditFormDifficultyHandler, set: func(renderer func(http.ResponseWriter, data.DifficultyPageData)) func() {
			previous := renderEditFormDifficultyPage
			renderEditFormDifficultyPage = renderer
			return func() { renderEditFormDifficultyPage = previous }
		}},
		{name: "delete", handler: DeleteFormDifficultyHandler, set: func(renderer func(http.ResponseWriter, data.DifficultyPageData)) func() {
			previous := renderDeleteFormDifficultyPage
			renderDeleteFormDifficultyPage = renderer
			return func() { renderDeleteFormDifficultyPage = previous }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var page data.DifficultyPageData
			restore := tc.set(func(_ http.ResponseWriter, got data.DifficultyPageData) { page = got })
			defer restore()
			response := serveAuthenticatedDifficultyRequest(t, http.MethodGet, "/?difficulty_id=1", nil, tc.handler, queries)
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d", response.Code)
			}
			if page.DifficultyContext.ID != 1 || page.DifficultyContext.Name != "owned" {
				t.Fatalf("context=%#v", page.DifficultyContext)
			}
			if page.CancelURL != data.DefaultQuestionRoutes.DifficultiesURL {
				t.Fatalf("CancelURL=%q", page.CancelURL)
			}
		})
	}
}

func TestDifficultyHandlersPreserveApostrophesAndQuotes(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		conn, queries := newDifficultyViewTestDB(t)
		name := `L'étude "complète"`
		response := serveAuthenticatedDifficultyRequest(t, http.MethodPost, "/", url.Values{"difficulty": {name}}, AddDifficultyHandler, queries)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d", response.Code)
		}
		assertStoredDifficultyName(t, conn, name)
	})
	t.Run("edit", func(t *testing.T) {
		conn, queries := newDifficultyViewTestDB(t)
		name := `L'étude "modifiée"`
		response := serveAuthenticatedDifficultyRequest(t, http.MethodPost, "/", url.Values{"difficulty_id": {"1"}, "new_difficulty": {name}}, EditDifficultyHandler, queries)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d", response.Code)
		}
		assertStoredDifficultyName(t, conn, name)
	})
}

func newDifficultyViewTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE difficulties(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL, UNIQUE(name,user_id));
		INSERT INTO difficulties(id,name,user_id) VALUES(1,'owned',1),(2,'foreign',2);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedDifficultyRequest(t *testing.T, method, target string, form url.Values, handler difficultyDBHandler, queries *db.Queries) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "text-reference-view-test-key-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	body := strings.NewReader("")
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	request := httptest.NewRequest(method, target, body)
	if form != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
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

func assertDifficultyURL(t *testing.T, rawURL, wantPath string, wantID int64) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != wantPath || parsed.Query().Get("difficulty_id") != strconv.FormatInt(wantID, 10) {
		t.Fatalf("URL=%q, want path=%q id=%d", rawURL, wantPath, wantID)
	}
}

func assertStoredDifficultyName(t *testing.T, conn *sql.DB, want string) {
	t.Helper()
	var got string
	if err := conn.QueryRow("SELECT name FROM difficulties WHERE name=? AND user_id=1", want).Scan(&got); err != nil || got != want {
		t.Fatalf("stored name=%q err=%v, want %q", got, err, want)
	}
}
