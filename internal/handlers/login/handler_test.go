package login

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/grapinou/LazyMarking/internal/db"
	"golang.org/x/crypto/bcrypt"
)

func setupLoginTest(t *testing.T, username string) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`CREATE TABLE users(id INTEGER PRIMARY KEY,username TEXT,email TEXT,hashpassword TEXT); INSERT INTO users VALUES(1,?,?,?)`, username, "alice@example.test", string(hash)); err != nil {
		t.Fatal(err)
	}
	initTestSessionStore(t)
	return conn, db.New(conn)
}

func TestLoggedHandlerAuthenticatesValidCredentialsAndReplacesPriorState(t *testing.T) {
	_, queries := setupLoginTest(t, "alice")
	form := url.Values{"username": {"alice"}, "password": {"correct-password"}}
	seedRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	priorCookie := saveTestSessionCookie(t, seedRequest, map[interface{}]interface{}{
		"user_id": int64(99), "username": "previous", "unexpected": "state",
	})
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(priorCookie)
	response := httptest.NewRecorder()
	LoggedHandler(response, request, queries)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Set-Cookie count=%d", len(cookies))
	}
	decodeRequest := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	decodeRequest.AddCookie(cookies[0])
	session, err := store.Get(decodeRequest, "session")
	if err != nil {
		t.Fatal(err)
	}
	if session.Values["user_id"] != int64(1) || session.Values["username"] != "alice" {
		t.Fatalf("session values=%v", session.Values)
	}
	if _, exists := session.Values["unexpected"]; exists {
		t.Fatalf("prior state survived login: %v", session.Values)
	}
}

func TestLoggedHandlerUsesSamePublicFailureForUnknownUserAndBadPassword(t *testing.T) {
	_, queries := setupLoginTest(t, "alice")
	var wantStatus int
	var wantBody string
	for i, tc := range []struct{ username, password string }{
		{"alice", "wrong-password"},
		{"unknown", "wrong-password"},
	} {
		form := url.Values{"username": {tc.username}, "password": {tc.password}}
		request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		LoggedHandler(response, request, queries)
		if i == 0 {
			wantStatus, wantBody = response.Code, response.Body.String()
		}
		if response.Code != wantStatus || response.Body.String() != wantBody || response.Code != http.StatusUnauthorized {
			t.Fatalf("case %d status/body=(%d,%q), want (%d,%q)", i, response.Code, response.Body.String(), wantStatus, wantBody)
		}
	}
}

func TestLoggedHandlerSessionSaveFailureDoesNotAuthenticate(t *testing.T) {
	_, queries := setupLoginTest(t, "alice")
	previous := saveLoginSession
	saveLoginSession = func(*sessions.Session, *http.Request, http.ResponseWriter) error {
		return errors.New("synthetic save failure")
	}
	t.Cleanup(func() { saveLoginSession = previous })
	form := url.Values{"username": {"alice"}, "password": {"correct-password"}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	LoggedHandler(response, request, queries)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", response.Code)
	}
	if len(response.Result().Cookies()) != 0 {
		t.Fatal("session-save failure emitted an authentication cookie")
	}
}
