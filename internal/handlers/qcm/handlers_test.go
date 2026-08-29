package qcm

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestTableQCMBuildsTypedItemsWithCountsAndOwnedURLs(t *testing.T) {
	_, queries := newQCMListHandlerTestDB(t)
	var page data.QCMPageData
	previous := renderTableQCMPage
	renderTableQCMPage = func(_ http.ResponseWriter, got data.QCMPageData) { page = got }
	defer func() { renderTableQCMPage = previous }()

	response := serveAuthenticatedQCMRequest(t, http.MethodGet, "/", nil, func(w http.ResponseWriter, r *http.Request) {
		TableQCMHandler(w, r, queries)
	})
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if len(page.QCMItems) != 2 {
		t.Fatalf("items count = %d, want 2", len(page.QCMItems))
	}
	wants := []struct {
		id    int64
		name  string
		count int64
	}{
		{id: 3, name: "owned empty", count: 0},
		{id: 1, name: "owned populated", count: 2},
	}
	for index, want := range wants {
		item := page.QCMItems[index]
		if item.ID != want.id || item.Name != want.name || item.QuestionCount != want.count {
			t.Fatalf("item %d = %#v, want id=%d name=%q count=%d", index, item, want.id, want.name, want.count)
		}
		urls := map[string]string{
			"composition":       item.CompositionURL,
			"preview":           item.PreviewURL,
			"preview landscape": item.PreviewLandscapeURL,
			"edit":              item.EditURL,
			"delete":            item.DeleteURL,
		}
		wantBases := map[string]string{
			"composition":       data.DefaultQCMRoutes.AddQuestionURL,
			"preview":           data.DefaultQCMRoutes.PreviewURL,
			"preview landscape": data.DefaultQCMRoutes.PreviewLandscapeURL,
			"edit":              data.DefaultQCMRoutes.EditURL,
			"delete":            data.DefaultQCMRoutes.DeleteURL,
		}
		for name, gotURL := range urls {
			if gotURL != data.QCMURL(wantBases[name], want.id) {
				t.Fatalf("item %d %s URL = %q", index, name, gotURL)
			}
		}
	}
}

func TestTableQCMBuildsNonNilEmptyList(t *testing.T) {
	conn, queries := newQCMListHandlerTestDB(t)
	if _, err := conn.Exec("DELETE FROM qcm WHERE user_id = 1"); err != nil {
		t.Fatal(err)
	}
	var page data.QCMPageData
	previous := renderTableQCMPage
	renderTableQCMPage = func(_ http.ResponseWriter, got data.QCMPageData) { page = got }
	defer func() { renderTableQCMPage = previous }()

	response := serveAuthenticatedQCMRequest(t, http.MethodGet, "/", nil, func(w http.ResponseWriter, r *http.Request) {
		TableQCMHandler(w, r, queries)
	})
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if page.QCMItems == nil || len(page.QCMItems) != 0 {
		t.Fatalf("QCMItems = %#v, want non-nil empty slice", page.QCMItems)
	}
}

func TestQCMHandlersReturnNotFoundForMissingOrForeignQCM(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`CREATE TABLE qcm (id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL); INSERT INTO qcm VALUES (1, 'owned', 1), (2, 'foreign', 2);`); err != nil {
		t.Fatal(err)
	}
	queries := db.New(conn)
	tests := []struct {
		name   string
		method string
		target string
		form   url.Values
		serve  http.HandlerFunc
	}{
		{name: "update missing", method: http.MethodPost, target: "/", form: url.Values{"qcm_id": {"999"}, "new_qcm": {"new"}}, serve: func(w http.ResponseWriter, r *http.Request) { EditQCMHandler(w, r, queries) }},
		{name: "delete foreign", method: http.MethodPost, target: "/", form: url.Values{"qcm_id": {"2"}}, serve: func(w http.ResponseWriter, r *http.Request) { DeleteQCMHandler(w, r, queries) }},
		{name: "edit form foreign", method: http.MethodGet, target: "/?qcm_id=2", serve: func(w http.ResponseWriter, r *http.Request) { EditFormQCMHandler(w, r, queries) }},
		{name: "delete form missing", method: http.MethodGet, target: "/?qcm_id=999", serve: func(w http.ResponseWriter, r *http.Request) { DeleteFormQCMHandler(w, r, queries) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedQCMRequest(t, tc.method, tc.target, tc.form, tc.serve)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
		})
	}
	var name string
	if err := conn.QueryRow("SELECT name FROM qcm WHERE id = 2 AND user_id = 2").Scan(&name); err != nil || name != "foreign" {
		t.Fatalf("foreign QCM name = %q, err = %v", name, err)
	}
}

func newQCMListHandlerTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE qcm (id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE qcm_questions (id INTEGER PRIMARY KEY, qcm_id INTEGER NOT NULL, question_id INTEGER NOT NULL, user_id INTEGER NOT NULL, position INTEGER NOT NULL);
		INSERT INTO qcm VALUES (1, 'owned populated', 1), (2, 'foreign', 2), (3, 'owned empty', 1);
		INSERT INTO qcm_questions VALUES
			(100, 1, 10, 1, 1),
			(101, 1, 11, 1, 2),
			(200, 2, 20, 2, 1);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedQCMRequest(t *testing.T, method, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "qcm-handler-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
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
