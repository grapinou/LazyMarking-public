package qcmpreview

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestPreviewHandlersRejectMissingAndForeignQCMBeforeConstruction(t *testing.T) {
	queries := newQCMPreviewTestQueries(t)
	buildCalls := 0
	previous := getPreviewQCMQuestions
	getPreviewQCMQuestions = func(_ int64, _ int64, _ *http.Request, _ *db.Queries) ([]config.Question, error) {
		buildCalls++
		return nil, nil
	}
	defer func() { getPreviewQCMQuestions = previous }()

	for _, qcmID := range []string{"999", "2"} {
		response := serveAuthenticatedPreviewRequest(t, "/?qcm_id="+qcmID, func(w http.ResponseWriter, r *http.Request) {
			PreviewQCMHandler(w, r, queries)
		})
		if response.Code != http.StatusNotFound {
			t.Fatalf("qcm_id=%s status=%d, want 404", qcmID, response.Code)
		}
	}
	if buildCalls != 0 {
		t.Fatalf("preview construction calls = %d, want 0", buildCalls)
	}
}

func TestPortraitAndLandscapeUseSharedReferenceConstruction(t *testing.T) {
	queries := newQCMPreviewTestQueries(t)
	buildCalls := 0
	previous := getPreviewQCMQuestions
	getPreviewQCMQuestions = func(userID, qcmID int64, _ *http.Request, _ *db.Queries) ([]config.Question, error) {
		buildCalls++
		if userID != 1 || qcmID != 1 {
			t.Fatalf("construction parent = user %d QCM %d, want user 1 QCM 1", userID, qcmID)
		}
		return nil, tools.ErrQuestionWithNoAnswer
	}
	defer func() { getPreviewQCMQuestions = previous }()

	handlers := []http.HandlerFunc{
		func(w http.ResponseWriter, r *http.Request) { PreviewQCMHandler(w, r, queries) },
		func(w http.ResponseWriter, r *http.Request) { PreviewQCMLandscapeHandler(w, r, queries) },
	}
	for _, handler := range handlers {
		response := serveAuthenticatedPreviewRequest(t, "/?qcm_id=1", handler)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want 303", response.Code)
		}
		if !strings.HasPrefix(response.Header().Get("Location"), data.ErrorMessageURL) {
			t.Fatalf("location = %q, want error route", response.Header().Get("Location"))
		}
	}
	if buildCalls != 2 {
		t.Fatalf("reference construction calls = %d, want 2", buildCalls)
	}
}

func TestOwnedEmptyQCMRedirectsInsteadOfReturningNotFound(t *testing.T) {
	queries := newQCMPreviewTestQueries(t)
	previous := getPreviewQCMQuestions
	getPreviewQCMQuestions = func(_ int64, _ int64, _ *http.Request, _ *db.Queries) ([]config.Question, error) {
		return []config.Question{}, nil
	}
	defer func() { getPreviewQCMQuestions = previous }()

	response := serveAuthenticatedPreviewRequest(t, "/?qcm_id=1", func(w http.ResponseWriter, r *http.Request) {
		PreviewQCMHandler(w, r, queries)
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("empty owned QCM status = %d, want 303", response.Code)
	}
	if !strings.HasPrefix(response.Header().Get("Location"), data.ErrorMessageURL) {
		t.Fatalf("location = %q, want error route", response.Header().Get("Location"))
	}
}

func newQCMPreviewTestQueries(t *testing.T) *db.Queries {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE qcm(id INTEGER PRIMARY KEY,name TEXT NOT NULL,user_id INTEGER NOT NULL);
		INSERT INTO qcm VALUES(1,'owned',1),(2,'foreign',2);
	`); err != nil {
		t.Fatal(err)
	}
	return db.New(conn)
}

func serveAuthenticatedPreviewRequest(t *testing.T, target string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "qcm-preview-handler-test-key-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, target, nil)
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
