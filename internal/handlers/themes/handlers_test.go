package themes

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

type themeDBHandler func(http.ResponseWriter, *http.Request, *db.Queries)

func TestThemeListViewData(t *testing.T) {
	conn, queries := newThemeViewTestDB(t)
	if _, err := conn.Exec("INSERT INTO themes(id,name,user_id) VALUES(3,'second',1)"); err != nil {
		t.Fatal(err)
	}
	var page data.ThemePageData
	previous := renderTableThemePage
	renderTableThemePage = func(_ http.ResponseWriter, got data.ThemePageData) { page = got }
	defer func() { renderTableThemePage = previous }()

	response := serveAuthenticatedThemeRequest(t, http.MethodGet, "/", nil, TableThemesHandler, queries)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", response.Code)
	}
	if len(page.ThemeItems) != 2 {
		t.Fatalf("items=%#v, want two", page.ThemeItems)
	}
	wantIDs := []int64{1, 3}
	wantNames := []string{"owned", "second"}
	for i, item := range page.ThemeItems {
		if item.ID != wantIDs[i] || item.Name != wantNames[i] {
			t.Fatalf("item %d=%#v", i, item)
		}
		assertThemeURL(t, item.EditURL, data.DefaultThemeRoutes.EditURL, item.ID)
		assertThemeURL(t, item.DeleteURL, data.DefaultThemeRoutes.DeleteURL, item.ID)
	}
	if _, err := conn.Exec("DELETE FROM themes WHERE user_id=1"); err != nil {
		t.Fatal(err)
	}
	page = data.ThemePageData{}
	serveAuthenticatedThemeRequest(t, http.MethodGet, "/", nil, TableThemesHandler, queries)
	if page.ThemeItems == nil || len(page.ThemeItems) != 0 {
		t.Fatalf("empty items=%#v, want non-nil empty slice", page.ThemeItems)
	}
}

func TestThemeFormsProvideContextAndCancelURL(t *testing.T) {
	_, queries := newThemeViewTestDB(t)
	t.Run("add", func(t *testing.T) {
		var page data.ThemePageData
		previous := renderAddThemeFormPage
		renderAddThemeFormPage = func(_ http.ResponseWriter, got data.ThemePageData) { page = got }
		defer func() { renderAddThemeFormPage = previous }()
		serveAuthenticatedThemeRequest(t, http.MethodGet, "/", nil, AddFormThemeHandler, queries)
		if page.CancelURL != data.DefaultQuestionRoutes.ThemesURL {
			t.Fatalf("CancelURL=%q", page.CancelURL)
		}
	})
	for _, tc := range []struct {
		name    string
		handler themeDBHandler
		set     func(func(http.ResponseWriter, data.ThemePageData)) func()
	}{
		{name: "edit", handler: EditFormThemeHandler, set: func(renderer func(http.ResponseWriter, data.ThemePageData)) func() {
			previous := renderEditFormThemePage
			renderEditFormThemePage = renderer
			return func() { renderEditFormThemePage = previous }
		}},
		{name: "delete", handler: DeleteFormThemeHandler, set: func(renderer func(http.ResponseWriter, data.ThemePageData)) func() {
			previous := renderDeleteFormThemePage
			renderDeleteFormThemePage = renderer
			return func() { renderDeleteFormThemePage = previous }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var page data.ThemePageData
			restore := tc.set(func(_ http.ResponseWriter, got data.ThemePageData) { page = got })
			defer restore()
			response := serveAuthenticatedThemeRequest(t, http.MethodGet, "/?theme_id=1", nil, tc.handler, queries)
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d", response.Code)
			}
			if page.ThemeContext.ID != 1 || page.ThemeContext.Name != "owned" {
				t.Fatalf("context=%#v", page.ThemeContext)
			}
			if page.CancelURL != data.DefaultQuestionRoutes.ThemesURL {
				t.Fatalf("CancelURL=%q", page.CancelURL)
			}
		})
	}
}

func TestThemeHandlersPreserveApostrophesAndQuotes(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		conn, queries := newThemeViewTestDB(t)
		name := `L'étude "complète"`
		response := serveAuthenticatedThemeRequest(t, http.MethodPost, "/", url.Values{"theme": {name}}, AddThemeHandler, queries)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d", response.Code)
		}
		assertStoredThemeName(t, conn, name)
	})
	t.Run("edit", func(t *testing.T) {
		conn, queries := newThemeViewTestDB(t)
		name := `L'étude "modifiée"`
		response := serveAuthenticatedThemeRequest(t, http.MethodPost, "/", url.Values{"theme_id": {"1"}, "new_theme": {name}}, EditThemeHandler, queries)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d", response.Code)
		}
		assertStoredThemeName(t, conn, name)
	})
}

func newThemeViewTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE themes(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL, UNIQUE(name,user_id));
		INSERT INTO themes(id,name,user_id) VALUES(1,'owned',1),(2,'foreign',2);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedThemeRequest(t *testing.T, method, target string, form url.Values, handler themeDBHandler, queries *db.Queries) *httptest.ResponseRecorder {
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

func assertThemeURL(t *testing.T, rawURL, wantPath string, wantID int64) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != wantPath || parsed.Query().Get("theme_id") != strconv.FormatInt(wantID, 10) {
		t.Fatalf("URL=%q, want path=%q id=%d", rawURL, wantPath, wantID)
	}
}

func assertStoredThemeName(t *testing.T, conn *sql.DB, want string) {
	t.Helper()
	var got string
	if err := conn.QueryRow("SELECT name FROM themes WHERE name=? AND user_id=1", want).Scan(&got); err != nil || got != want {
		t.Fatalf("stored name=%q err=%v, want %q", got, err, want)
	}
}
