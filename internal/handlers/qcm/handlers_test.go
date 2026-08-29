package qcm

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/mattn/go-sqlite3"
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
	if _, err := conn.Exec(`CREATE TABLE qcm (id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL); CREATE TABLE exams (id INTEGER PRIMARY KEY, qcm_id INTEGER NOT NULL, user_id INTEGER NOT NULL); INSERT INTO qcm VALUES (1, 'owned', 1), (2, 'foreign', 2);`); err != nil {
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

func TestDeleteQCMHandlerAppliesOwnedCascadeAndProtectedExamSemantics(t *testing.T) {
	tests := []struct {
		name       string
		qcmID      string
		wantStatus int
		assert     func(*testing.T, *sql.DB, *httptest.ResponseRecorder)
	}{
		{
			name: "deletable composed QCM", qcmID: "1", wantStatus: http.StatusSeeOther,
			assert: func(t *testing.T, conn *sql.DB, response *httptest.ResponseRecorder) {
				if response.Header().Get("Location") != data.DefaultDashboardRoutes.QcmURL {
					t.Fatalf("Location = %q", response.Header().Get("Location"))
				}
				assertQCMHandlerCount(t, conn, "qcm", "id=1", 0)
				assertQCMHandlerCount(t, conn, "qcm_questions", "qcm_id=1", 0)
				assertQCMHandlerCount(t, conn, "questions", "id=10", 1)
			},
		},
		{
			name: "missing QCM", qcmID: "999", wantStatus: http.StatusNotFound,
			assert: func(t *testing.T, conn *sql.DB, _ *httptest.ResponseRecorder) {
				assertQCMHandlerCount(t, conn, "qcm", "id=1", 1)
			},
		},
		{
			name: "foreign QCM", qcmID: "2", wantStatus: http.StatusNotFound,
			assert: func(t *testing.T, conn *sql.DB, _ *httptest.ResponseRecorder) {
				assertQCMHandlerCount(t, conn, "qcm", "id=2 AND user_id=2", 1)
			},
		},
		{
			name: "exam-protected QCM", qcmID: "3", wantStatus: http.StatusSeeOther,
			assert: func(t *testing.T, conn *sql.DB, response *httptest.ResponseRecorder) {
				assertProtectedQCMRedirect(t, response)
				assertQCMHandlerCount(t, conn, "qcm", "id=3", 1)
				assertQCMHandlerCount(t, conn, "qcm_questions", "qcm_id=3", 1)
				assertQCMHandlerCount(t, conn, "exams", "qcm_id=3", 1)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, queries := newQCMDeleteHandlerTestDB(t)
			form := url.Values{"qcm_id": {tc.qcmID}}
			response := serveAuthenticatedQCMRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
				DeleteQCMHandler(w, r, queries)
			})
			if response.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tc.wantStatus)
			}
			tc.assert(t, conn, response)
		})
	}
}

func TestDeleteQCMHandlerTranslatesRaceForeignKeyConstraint(t *testing.T) {
	conn, queries := newQCMDeleteHandlerTestDB(t)
	previous := deleteOwnedQCM
	deleteOwnedQCM = func(_ *db.Queries, _ context.Context, _ db.DeleteQCMParams) (int64, error) {
		return 0, sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintForeignKey}
	}
	defer func() { deleteOwnedQCM = previous }()

	form := url.Values{"qcm_id": {"1"}}
	response := serveAuthenticatedQCMRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
		DeleteQCMHandler(w, r, queries)
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", response.Code)
	}
	assertProtectedQCMRedirect(t, response)
	assertQCMHandlerCount(t, conn, "qcm", "id=1", 1)
	assertQCMHandlerCount(t, conn, "qcm_questions", "qcm_id=1", 1)
}

func TestDeleteQCMHandlerProtectedPrecheckSkipsDelete(t *testing.T) {
	conn, queries := newQCMDeleteHandlerTestDB(t)
	called := false
	previous := deleteOwnedQCM
	deleteOwnedQCM = func(_ *db.Queries, _ context.Context, _ db.DeleteQCMParams) (int64, error) {
		called = true
		return 1, nil
	}
	defer func() { deleteOwnedQCM = previous }()

	form := url.Values{"qcm_id": {"3"}}
	response := serveAuthenticatedQCMRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
		DeleteQCMHandler(w, r, queries)
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", response.Code)
	}
	if called {
		t.Fatal("DeleteQCM was called for an exam-protected QCM")
	}
	assertProtectedQCMRedirect(t, response)
	assertQCMHandlerCount(t, conn, "qcm", "id=3", 1)
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

func newQCMDeleteHandlerTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE qcm(id INTEGER PRIMARY KEY,name TEXT NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE questions(id INTEGER PRIMARY KEY,content TEXT NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE qcm_questions(
			id INTEGER PRIMARY KEY,qcm_id INTEGER NOT NULL REFERENCES qcm(id) ON DELETE CASCADE,
			question_id INTEGER NOT NULL REFERENCES questions(id) ON DELETE RESTRICT,
			user_id INTEGER NOT NULL,position INTEGER NOT NULL);
		CREATE TABLE exams(id INTEGER PRIMARY KEY,qcm_id INTEGER NOT NULL REFERENCES qcm(id) ON DELETE RESTRICT,user_id INTEGER NOT NULL);
		INSERT INTO qcm VALUES(1,'deletable',1),(2,'foreign',2),(3,'protected',1);
		INSERT INTO questions VALUES(10,'shared bank question',1),(20,'foreign question',2);
		INSERT INTO qcm_questions VALUES(100,1,10,1,1),(200,2,20,2,1),(300,3,10,1,1);
		INSERT INTO exams VALUES(30,3,1);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func assertProtectedQCMRedirect(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Path != data.ErrorMessageURL {
		t.Fatalf("redirect path = %q, want %q", location.Path, data.ErrorMessageURL)
	}
	if message := location.Query().Get("errormessage"); message != "Ce QCM est utilisé par une évaluation et ne peut pas être supprimé." {
		t.Fatalf("error message = %q", message)
	}
}

func assertQCMHandlerCount(t *testing.T, conn *sql.DB, table, condition string, want int) {
	t.Helper()
	var got int
	if err := conn.QueryRow("SELECT COUNT(*) FROM " + table + " WHERE " + condition).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s where %s count=%d, want %d", table, condition, got, want)
	}
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
