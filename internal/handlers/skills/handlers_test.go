package skills

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

type skillDBHandler func(http.ResponseWriter, *http.Request, *db.Queries)

func TestSkillListViewData(t *testing.T) {
	conn, queries := newSkillViewTestDB(t)
	if _, err := conn.Exec("INSERT INTO skills(id,name,user_id) VALUES(3,'second',1)"); err != nil {
		t.Fatal(err)
	}
	var page data.SkillPageData
	previous := renderTableSkillPage
	renderTableSkillPage = func(_ http.ResponseWriter, got data.SkillPageData) { page = got }
	defer func() { renderTableSkillPage = previous }()

	response := serveAuthenticatedSkillRequest(t, http.MethodGet, "/", nil, TableSkillsHandler, queries)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", response.Code)
	}
	if len(page.SkillItems) != 2 {
		t.Fatalf("items=%#v, want two", page.SkillItems)
	}
	wantIDs := []int64{1, 3}
	wantNames := []string{"owned", "second"}
	for i, item := range page.SkillItems {
		if item.ID != wantIDs[i] || item.Name != wantNames[i] {
			t.Fatalf("item %d=%#v", i, item)
		}
		assertSkillURL(t, item.EditURL, data.DefaultSkillRoutes.EditURL, item.ID)
		assertSkillURL(t, item.DeleteURL, data.DefaultSkillRoutes.DeleteURL, item.ID)
	}
	if _, err := conn.Exec("DELETE FROM skills WHERE user_id=1"); err != nil {
		t.Fatal(err)
	}
	page = data.SkillPageData{}
	serveAuthenticatedSkillRequest(t, http.MethodGet, "/", nil, TableSkillsHandler, queries)
	if page.SkillItems == nil || len(page.SkillItems) != 0 {
		t.Fatalf("empty items=%#v, want non-nil empty slice", page.SkillItems)
	}
}

func TestSkillFormsProvideContextAndCancelURL(t *testing.T) {
	_, queries := newSkillViewTestDB(t)
	t.Run("add", func(t *testing.T) {
		var page data.SkillPageData
		previous := renderAddFormSkillPage
		renderAddFormSkillPage = func(_ http.ResponseWriter, got data.SkillPageData) { page = got }
		defer func() { renderAddFormSkillPage = previous }()
		serveAuthenticatedSkillRequest(t, http.MethodGet, "/", nil, AddFormSkillHandler, queries)
		if page.CancelURL != data.DefaultQuestionRoutes.SkillsURL {
			t.Fatalf("CancelURL=%q", page.CancelURL)
		}
	})
	for _, tc := range []struct {
		name    string
		handler skillDBHandler
		set     func(func(http.ResponseWriter, data.SkillPageData)) func()
	}{
		{name: "edit", handler: EditFormSkillHandler, set: func(renderer func(http.ResponseWriter, data.SkillPageData)) func() {
			previous := renderEditFormSkillPage
			renderEditFormSkillPage = renderer
			return func() { renderEditFormSkillPage = previous }
		}},
		{name: "delete", handler: DeleteFormSkillHandler, set: func(renderer func(http.ResponseWriter, data.SkillPageData)) func() {
			previous := renderDeleteFormSkillPage
			renderDeleteFormSkillPage = renderer
			return func() { renderDeleteFormSkillPage = previous }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var page data.SkillPageData
			restore := tc.set(func(_ http.ResponseWriter, got data.SkillPageData) { page = got })
			defer restore()
			response := serveAuthenticatedSkillRequest(t, http.MethodGet, "/?skill_id=1", nil, tc.handler, queries)
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d", response.Code)
			}
			if page.SkillContext.ID != 1 || page.SkillContext.Name != "owned" {
				t.Fatalf("context=%#v", page.SkillContext)
			}
			if page.CancelURL != data.DefaultQuestionRoutes.SkillsURL {
				t.Fatalf("CancelURL=%q", page.CancelURL)
			}
		})
	}
}

func TestSkillHandlersPreserveApostrophesAndQuotes(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		conn, queries := newSkillViewTestDB(t)
		name := `L'étude "complète"`
		response := serveAuthenticatedSkillRequest(t, http.MethodPost, "/", url.Values{"skill": {name}}, AddSkillHandler, queries)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d", response.Code)
		}
		assertStoredSkillName(t, conn, name)
	})
	t.Run("edit", func(t *testing.T) {
		conn, queries := newSkillViewTestDB(t)
		name := `L'étude "modifiée"`
		response := serveAuthenticatedSkillRequest(t, http.MethodPost, "/", url.Values{"skill_id": {"1"}, "new_skill": {name}}, EditSkillHandler, queries)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d", response.Code)
		}
		assertStoredSkillName(t, conn, name)
	})
}

func newSkillViewTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE skills(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL, UNIQUE(name,user_id));
		INSERT INTO skills(id,name,user_id) VALUES(1,'owned',1),(2,'foreign',2);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedSkillRequest(t *testing.T, method, target string, form url.Values, handler skillDBHandler, queries *db.Queries) *httptest.ResponseRecorder {
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

func assertSkillURL(t *testing.T, rawURL, wantPath string, wantID int64) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != wantPath || parsed.Query().Get("skill_id") != strconv.FormatInt(wantID, 10) {
		t.Fatalf("URL=%q, want path=%q id=%d", rawURL, wantPath, wantID)
	}
}

func assertStoredSkillName(t *testing.T, conn *sql.DB, want string) {
	t.Helper()
	var got string
	if err := conn.QueryRow("SELECT name FROM skills WHERE name=? AND user_id=1", want).Scan(&got); err != nil || got != want {
		t.Fatalf("stored name=%q err=%v, want %q", got, err, want)
	}
}
