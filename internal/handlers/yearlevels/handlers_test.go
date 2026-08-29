package yearlevels

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

type yearlevelDBHandler func(http.ResponseWriter, *http.Request, *db.Queries)

func TestYearLevelListViewData(t *testing.T) {
	conn, queries := newYearLevelViewTestDB(t)
	if _, err := conn.Exec("INSERT INTO year_levels(id,name,user_id) VALUES(3,'second',1)"); err != nil {
		t.Fatal(err)
	}
	var page data.YearLevelPageData
	previous := renderTableYearLevelPage
	renderTableYearLevelPage = func(_ http.ResponseWriter, got data.YearLevelPageData) { page = got }
	defer func() { renderTableYearLevelPage = previous }()

	response := serveAuthenticatedYearLevelRequest(t, http.MethodGet, "/", nil, TableYearLevelsHandler, queries)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", response.Code)
	}
	if len(page.YearLevelItems) != 2 {
		t.Fatalf("items=%#v, want two", page.YearLevelItems)
	}
	wantIDs := []int64{1, 3}
	wantNames := []string{"owned", "second"}
	for i, item := range page.YearLevelItems {
		if item.ID != wantIDs[i] || item.Name != wantNames[i] {
			t.Fatalf("item %d=%#v", i, item)
		}
		assertYearLevelURL(t, item.EditURL, data.DefaultYearLevelRoutes.EditURL, item.ID)
		assertYearLevelURL(t, item.DeleteURL, data.DefaultYearLevelRoutes.DeleteURL, item.ID)
	}
	if _, err := conn.Exec("DELETE FROM year_levels WHERE user_id=1"); err != nil {
		t.Fatal(err)
	}
	page = data.YearLevelPageData{}
	serveAuthenticatedYearLevelRequest(t, http.MethodGet, "/", nil, TableYearLevelsHandler, queries)
	if page.YearLevelItems == nil || len(page.YearLevelItems) != 0 {
		t.Fatalf("empty items=%#v, want non-nil empty slice", page.YearLevelItems)
	}
}

func TestYearLevelFormsProvideContextAndCancelURL(t *testing.T) {
	_, queries := newYearLevelViewTestDB(t)
	t.Run("add", func(t *testing.T) {
		var page data.YearLevelPageData
		previous := renderAddFormYearLevelPage
		renderAddFormYearLevelPage = func(_ http.ResponseWriter, got data.YearLevelPageData) { page = got }
		defer func() { renderAddFormYearLevelPage = previous }()
		serveAuthenticatedYearLevelRequest(t, http.MethodGet, "/", nil, AddFormYearLevelHandler, queries)
		if page.CancelURL != data.DefaultQuestionRoutes.YearLevelsURL {
			t.Fatalf("CancelURL=%q", page.CancelURL)
		}
	})
	for _, tc := range []struct {
		name    string
		handler yearlevelDBHandler
		set     func(func(http.ResponseWriter, data.YearLevelPageData)) func()
	}{
		{name: "edit", handler: EditFormYearLevelHandler, set: func(renderer func(http.ResponseWriter, data.YearLevelPageData)) func() {
			previous := renderEditFormYearLevelPage
			renderEditFormYearLevelPage = renderer
			return func() { renderEditFormYearLevelPage = previous }
		}},
		{name: "delete", handler: DeleteFormYearLevelHandler, set: func(renderer func(http.ResponseWriter, data.YearLevelPageData)) func() {
			previous := renderDeleteFormYearLevelPage
			renderDeleteFormYearLevelPage = renderer
			return func() { renderDeleteFormYearLevelPage = previous }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var page data.YearLevelPageData
			restore := tc.set(func(_ http.ResponseWriter, got data.YearLevelPageData) { page = got })
			defer restore()
			response := serveAuthenticatedYearLevelRequest(t, http.MethodGet, "/?yearlevel_id=1", nil, tc.handler, queries)
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d", response.Code)
			}
			if page.YearLevelContext.ID != 1 || page.YearLevelContext.Name != "owned" {
				t.Fatalf("context=%#v", page.YearLevelContext)
			}
			if page.CancelURL != data.DefaultQuestionRoutes.YearLevelsURL {
				t.Fatalf("CancelURL=%q", page.CancelURL)
			}
		})
	}
}

func TestYearLevelHandlersPreserveApostrophesAndQuotes(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		conn, queries := newYearLevelViewTestDB(t)
		name := `L'étude "complète"`
		response := serveAuthenticatedYearLevelRequest(t, http.MethodPost, "/", url.Values{"yearlevel": {name}}, AddYearLevelHandler, queries)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d", response.Code)
		}
		assertStoredYearLevelName(t, conn, name)
	})
	t.Run("edit", func(t *testing.T) {
		conn, queries := newYearLevelViewTestDB(t)
		name := `L'étude "modifiée"`
		response := serveAuthenticatedYearLevelRequest(t, http.MethodPost, "/", url.Values{"yearlevel_id": {"1"}, "new_yearlevel": {name}}, EditYearLevelHandler, queries)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d", response.Code)
		}
		assertStoredYearLevelName(t, conn, name)
	})
}

func newYearLevelViewTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE year_levels(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL, UNIQUE(name,user_id));
		INSERT INTO year_levels(id,name,user_id) VALUES(1,'owned',1),(2,'foreign',2);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedYearLevelRequest(t *testing.T, method, target string, form url.Values, handler yearlevelDBHandler, queries *db.Queries) *httptest.ResponseRecorder {
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

func assertYearLevelURL(t *testing.T, rawURL, wantPath string, wantID int64) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != wantPath || parsed.Query().Get("yearlevel_id") != strconv.FormatInt(wantID, 10) {
		t.Fatalf("URL=%q, want path=%q id=%d", rawURL, wantPath, wantID)
	}
}

func assertStoredYearLevelName(t *testing.T, conn *sql.DB, want string) {
	t.Helper()
	var got string
	if err := conn.QueryRow("SELECT name FROM year_levels WHERE name=? AND user_id=1", want).Scan(&got); err != nil || got != want {
		t.Fatalf("stored name=%q err=%v, want %q", got, err, want)
	}
}
