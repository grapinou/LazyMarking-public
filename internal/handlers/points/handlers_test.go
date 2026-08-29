package points

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
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestAddPointHandlerEnforcesPositiveContractWithoutUpperLimit(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantStatus int
		wantStored bool
	}{
		{name: "one", value: "1", wantStatus: http.StatusSeeOther, wantStored: true},
		{name: "one hundred", value: "100", wantStatus: http.StatusSeeOther, wantStored: true},
		{name: "above UI range", value: "101", wantStatus: http.StatusSeeOther, wantStored: true},
		{name: "zero", value: "0", wantStatus: http.StatusBadRequest},
		{name: "negative", value: "-1", wantStatus: http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, queries := newPointHandlerTestDB(t)
			response := serveAuthenticatedPointRequest(t, http.MethodPost, url.Values{"point": {tc.value}}, func(w http.ResponseWriter, r *http.Request) {
				AddPointHandler(w, r, queries)
			})
			if response.Code != tc.wantStatus {
				t.Fatalf("status=%d, want %d", response.Code, tc.wantStatus)
			}
			var count int
			if err := conn.QueryRow("SELECT COUNT(*) FROM points WHERE point_value=? AND user_id=1", tc.value).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if (count == 1) != tc.wantStored {
				t.Fatalf("stored count=%d, wantStored=%t", count, tc.wantStored)
			}
		})
	}
}

func TestPointListViewDataAndEmptyState(t *testing.T) {
	conn, queries := newPointHandlerTestDB(t)
	var page data.PointPageData
	previous := renderTablePointPage
	renderTablePointPage = func(_ http.ResponseWriter, got data.PointPageData) { page = got }
	defer func() { renderTablePointPage = previous }()

	response := serveAuthenticatedPointRequest(t, http.MethodGet, nil, func(w http.ResponseWriter, r *http.Request) {
		TablePointsHandler(w, r, queries)
	})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", response.Code)
	}
	if len(page.PointItems) != 2 {
		t.Fatalf("items=%#v, want two owned values", page.PointItems)
	}
	wantValues := []int64{5, 101}
	wantIDs := []int64{1, 2}
	for i, item := range page.PointItems {
		if item.ID != wantIDs[i] || item.Value != wantValues[i] {
			t.Fatalf("item %d=%#v", i, item)
		}
		assertPointItemURL(t, item.EditURL, data.DefaultPointRoutes.EditURL, item.ID)
		assertPointItemURL(t, item.DeleteURL, data.DefaultPointRoutes.DeleteURL, item.ID)
	}

	if _, err := conn.Exec("DELETE FROM points WHERE user_id=1"); err != nil {
		t.Fatal(err)
	}
	page = data.PointPageData{}
	serveAuthenticatedPointRequest(t, http.MethodGet, nil, func(w http.ResponseWriter, r *http.Request) {
		TablePointsHandler(w, r, queries)
	})
	if page.PointItems == nil || len(page.PointItems) != 0 {
		t.Fatalf("empty items=%#v, want non-nil empty slice", page.PointItems)
	}
}

func TestPointAddAndDeleteFormsProvideTypedData(t *testing.T) {
	_, queries := newPointHandlerTestDB(t)

	t.Run("add", func(t *testing.T) {
		var page data.PointPageData
		previous := renderAddFormPointPage
		renderAddFormPointPage = func(_ http.ResponseWriter, got data.PointPageData) { page = got }
		defer func() { renderAddFormPointPage = previous }()
		serveAuthenticatedPointRequest(t, http.MethodGet, nil, func(w http.ResponseWriter, r *http.Request) {
			AddFormPointHandler(w, r, queries)
		})
		if page.CancelURL != data.DefaultQuestionRoutes.PointsURL {
			t.Fatalf("CancelURL=%q", page.CancelURL)
		}
	})

	t.Run("delete", func(t *testing.T) {
		var page data.PointPageData
		previous := renderDeleteFormPointPage
		renderDeleteFormPointPage = func(_ http.ResponseWriter, got data.PointPageData) { page = got }
		defer func() { renderDeleteFormPointPage = previous }()
		response := serveAuthenticatedPointRequest(t, http.MethodGet, url.Values{"point_id": {"1"}}, func(w http.ResponseWriter, r *http.Request) {
			DeleteFormPointHandler(w, r, queries)
		})
		if response.Code != http.StatusOK {
			t.Fatalf("status=%d, want 200", response.Code)
		}
		if page.PointContext.ID != 1 || page.PointContext.Value != 5 {
			t.Fatalf("context=%#v", page.PointContext)
		}
		if page.CancelURL != data.DefaultQuestionRoutes.PointsURL {
			t.Fatalf("CancelURL=%q", page.CancelURL)
		}
	})
}

func TestEditFormPointProvidesCurrentValueWithoutArtificialRange(t *testing.T) {
	for _, pointID := range []string{"1", "2"} {
		t.Run(pointID, func(t *testing.T) {
			_, queries := newPointHandlerTestDB(t)
			var page data.PointPageData
			previous := renderEditFormPointPage
			renderEditFormPointPage = func(_ http.ResponseWriter, got data.PointPageData) { page = got }
			defer func() { renderEditFormPointPage = previous }()

			response := serveAuthenticatedPointRequest(t, http.MethodGet, url.Values{"point_id": {pointID}}, func(w http.ResponseWriter, r *http.Request) {
				EditFormPointHandler(w, r, queries)
			})
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d, want 200", response.Code)
			}
			wantValue := int64(5)
			if pointID == "2" {
				wantValue = 101
			}
			if page.Form.ID != mustParsePointID(t, pointID) || page.Form.CurrentValue != wantValue {
				t.Fatalf("Form=%#v, want ID %s current %d", page.Form, pointID, wantValue)
			}
			if page.CancelURL != data.DefaultQuestionRoutes.PointsURL {
				t.Fatalf("CancelURL=%q", page.CancelURL)
			}
		})
	}
}

func TestEditPointHandlerPreservesUnchangedValueAndRejectsNonPositiveValues(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantStatus int
	}{
		{name: "unchanged", value: "5", wantStatus: http.StatusSeeOther},
		{name: "zero", value: "0", wantStatus: http.StatusBadRequest},
		{name: "negative", value: "-1", wantStatus: http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, queries := newPointHandlerTestDB(t)
			form := url.Values{"point_id": {"1"}, "new_point": {tc.value}}
			response := serveAuthenticatedPointRequest(t, http.MethodPost, form, func(w http.ResponseWriter, r *http.Request) {
				EditPointHandler(w, r, queries)
			})
			if response.Code != tc.wantStatus {
				t.Fatalf("status=%d, want %d", response.Code, tc.wantStatus)
			}
			var value int64
			if err := conn.QueryRow("SELECT point_value FROM points WHERE id=1").Scan(&value); err != nil {
				t.Fatal(err)
			}
			if value != 5 {
				t.Fatalf("stored value=%d, want unchanged 5", value)
			}
		})
	}
}

func newPointHandlerTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE points(
			id INTEGER PRIMARY KEY,
			point_value INTEGER NOT NULL CHECK(point_value >= 1),
			user_id INTEGER NOT NULL,
			UNIQUE(point_value,user_id)
		);
		INSERT INTO points VALUES(1,5,1),(2,101,1),(3,7,2);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedPointRequest(t *testing.T, method string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "point-handler-test-key-long-enough")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	target := "/"
	var body *strings.Reader
	if method == http.MethodGet {
		target += "?" + form.Encode()
		body = strings.NewReader("")
	} else {
		body = strings.NewReader(form.Encode())
	}
	request := httptest.NewRequest(method, target, body)
	if method == http.MethodPost {
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

func mustParsePointID(t *testing.T, value string) int64 {
	t.Helper()
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func assertPointItemURL(t *testing.T, rawURL, wantPath string, wantID int64) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != wantPath || parsed.Query().Get("point_id") != strconv.FormatInt(wantID, 10) {
		t.Fatalf("URL=%q, want path=%q point_id=%d", rawURL, wantPath, wantID)
	}
}
