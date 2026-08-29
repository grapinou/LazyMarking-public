package subjects

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

type subjectDBHandler func(http.ResponseWriter, *http.Request, *db.Queries)

func TestTableSubjectsHandlerBuildsTypedItemsAndEmptyState(t *testing.T) {
	conn, queries := newSubjectHandlerTestDB(t)
	if _, err := conn.Exec("INSERT INTO subjects(id,name,user_id) VALUES(3,'second',1)"); err != nil {
		t.Fatal(err)
	}

	var page data.SubjectPageData
	previous := renderTableSubjectPage
	renderTableSubjectPage = func(_ http.ResponseWriter, got data.SubjectPageData) { page = got }
	defer func() { renderTableSubjectPage = previous }()

	response := serveAuthenticatedSubjectRequest(t, http.MethodGet, "/", nil, TableSubjectsHandler, queries)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", response.Code)
	}
	if len(page.SubjectItems) != 2 {
		t.Fatalf("items=%#v, want two owned subjects", page.SubjectItems)
	}
	want := []struct {
		id   int64
		name string
	}{{id: 1, name: "owned"}, {id: 3, name: "second"}}
	for index, item := range page.SubjectItems {
		if item.ID != want[index].id || item.Name != want[index].name {
			t.Fatalf("item %d=%#v, want ID=%d Name=%q", index, item, want[index].id, want[index].name)
		}
		assertSubjectItemURL(t, item.EditURL, data.DefaultSubjectRoutes.EditURL, item.ID)
		assertSubjectItemURL(t, item.DeleteURL, data.DefaultSubjectRoutes.DeleteURL, item.ID)
	}

	if _, err := conn.Exec("DELETE FROM subjects WHERE user_id=1"); err != nil {
		t.Fatal(err)
	}
	page = data.SubjectPageData{}
	serveAuthenticatedSubjectRequest(t, http.MethodGet, "/", nil, TableSubjectsHandler, queries)
	if page.SubjectItems == nil || len(page.SubjectItems) != 0 {
		t.Fatalf("empty items=%#v, want a non-nil empty slice", page.SubjectItems)
	}
}

func TestSubjectFormsProvideTypedContextAndCancelURL(t *testing.T) {
	_, queries := newSubjectHandlerTestDB(t)

	t.Run("add", func(t *testing.T) {
		var page data.SubjectPageData
		previous := renderAddFormSubjectPage
		renderAddFormSubjectPage = func(_ http.ResponseWriter, got data.SubjectPageData) { page = got }
		defer func() { renderAddFormSubjectPage = previous }()
		serveAuthenticatedSubjectRequest(t, http.MethodGet, "/", nil, AddFormSubjectHandler, queries)
		if page.CancelURL != data.DefaultQuestionRoutes.SubjectsURL {
			t.Fatalf("CancelURL=%q", page.CancelURL)
		}
	})

	for _, tc := range []struct {
		name    string
		handler subjectDBHandler
		set     func(func(http.ResponseWriter, data.SubjectPageData)) func()
	}{
		{name: "edit", handler: EditFormSubjectHandler, set: func(renderer func(http.ResponseWriter, data.SubjectPageData)) func() {
			previous := renderEditFormSubjectPage
			renderEditFormSubjectPage = renderer
			return func() { renderEditFormSubjectPage = previous }
		}},
		{name: "delete", handler: DeleteFormSubjectHandler, set: func(renderer func(http.ResponseWriter, data.SubjectPageData)) func() {
			previous := renderDeleteFormSubjectPage
			renderDeleteFormSubjectPage = renderer
			return func() { renderDeleteFormSubjectPage = previous }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var page data.SubjectPageData
			restore := tc.set(func(_ http.ResponseWriter, got data.SubjectPageData) { page = got })
			defer restore()
			response := serveAuthenticatedSubjectRequest(t, http.MethodGet, "/?subject_id=1", nil, tc.handler, queries)
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d, want 200", response.Code)
			}
			if page.SubjectContext.ID != 1 || page.SubjectContext.Name != "owned" {
				t.Fatalf("context=%#v", page.SubjectContext)
			}
			if page.CancelURL != data.DefaultQuestionRoutes.SubjectsURL {
				t.Fatalf("CancelURL=%q", page.CancelURL)
			}
		})
	}
}

func TestSubjectHandlersPreserveApostrophesAndQuotes(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		conn, queries := newSubjectHandlerTestDB(t)
		name := `L'étude "physique"`
		response := serveAuthenticatedSubjectRequest(t, http.MethodPost, "/", url.Values{"subject": {name}}, AddSubjectHandler, queries)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d, want 303", response.Code)
		}
		assertStoredSubjectName(t, conn, 1, name)
	})

	t.Run("edit", func(t *testing.T) {
		conn, queries := newSubjectHandlerTestDB(t)
		name := `L'étude "des forces"`
		response := serveAuthenticatedSubjectRequest(t, http.MethodPost, "/", url.Values{"subject_id": {"1"}, "new_subject": {name}}, EditSubjectHandler, queries)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d, want 303", response.Code)
		}
		assertStoredSubjectName(t, conn, 1, name)
	})
}

func TestOwnedSubjectHandlersDoNotReportZeroRowsAsSuccess(t *testing.T) {
	conn, queries := newSubjectHandlerTestDB(t)
	tests := []struct {
		name    string
		handler subjectDBHandler
		form    url.Values
	}{
		{name: "update missing", handler: EditSubjectHandler, form: url.Values{"subject_id": {"999"}, "new_subject": {"changed"}}},
		{name: "update foreign owned", handler: EditSubjectHandler, form: url.Values{"subject_id": {"2"}, "new_subject": {"changed"}}},
		{name: "delete missing", handler: DeleteSubjectHandler, form: url.Values{"subject_id": {"999"}}},
		{name: "delete foreign owned", handler: DeleteSubjectHandler, form: url.Values{"subject_id": {"2"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedSubjectRequest(t, http.MethodPost, "/", tc.form, tc.handler, queries)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
		})
	}

	var foreignName string
	if err := conn.QueryRow("SELECT name FROM subjects WHERE id = 2 AND user_id = 2").Scan(&foreignName); err != nil || foreignName != "foreign" {
		t.Fatalf("foreign-owned subject = %q, err = %v; want unchanged", foreignName, err)
	}
}

func TestOwnedSubjectFormsReturnNotFound(t *testing.T) {
	_, queries := newSubjectHandlerTestDB(t)
	tests := []struct {
		name    string
		handler subjectDBHandler
		query   string
	}{
		{name: "edit missing", handler: EditFormSubjectHandler, query: "subject_id=999"},
		{name: "edit foreign owned", handler: EditFormSubjectHandler, query: "subject_id=2"},
		{name: "delete missing", handler: DeleteFormSubjectHandler, query: "subject_id=999"},
		{name: "delete foreign owned", handler: DeleteFormSubjectHandler, query: "subject_id=2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedSubjectRequest(t, http.MethodGet, "/?"+tc.query, nil, tc.handler, queries)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
		})
	}
}

func newSubjectHandlerTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE subjects (id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL, UNIQUE(name, user_id));
		INSERT INTO subjects (id, name, user_id) VALUES (1, 'owned', 1), (2, 'foreign', 2);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedSubjectRequest(t *testing.T, method, target string, form url.Values, handler subjectDBHandler, queries *db.Queries) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "reference-handler-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}

	var body *strings.Reader
	if form == nil {
		body = strings.NewReader("")
	} else {
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
	session.Values["username"] = "test-user"
	cookieResponse := httptest.NewRecorder()
	if err := session.Save(request, cookieResponse); err != nil {
		t.Fatal(err)
	}
	for _, cookie := range cookieResponse.Result().Cookies() {
		request.AddCookie(cookie)
	}

	response := httptest.NewRecorder()
	login.CheckAuth(tools.HandlerWithDB(handler, queries)).ServeHTTP(response, request)
	return response
}

func assertSubjectItemURL(t *testing.T, rawURL, wantPath string, wantID int64) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != wantPath || parsed.Query().Get("subject_id") != strconv.FormatInt(wantID, 10) {
		t.Fatalf("URL=%q, want path %q subject_id %d", rawURL, wantPath, wantID)
	}
}

func assertStoredSubjectName(t *testing.T, conn *sql.DB, userID int64, want string) {
	t.Helper()
	var got string
	if err := conn.QueryRow("SELECT name FROM subjects WHERE name=? AND user_id=?", want, userID).Scan(&got); err != nil || got != want {
		t.Fatalf("stored name=%q err=%v, want %q", got, err, want)
	}
}
