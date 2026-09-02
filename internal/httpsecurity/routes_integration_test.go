package httpsecurity_test

import (
	"bytes"
	"context"
	"database/sql"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/csrf"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/logout"
	"github.com/grapinou/LazyMarking/internal/handlers/marking"
	"github.com/grapinou/LazyMarking/internal/handlers/subjects"
	"github.com/grapinou/LazyMarking/internal/httpsecurity"
	"golang.org/x/crypto/bcrypt"
)

var integrationTokenPattern = regexp.MustCompile(`name="gorilla\.csrf\.Token" value="([^"]+)"`)

func TestLoginAndLogoutRoutesUseGlobalCSRFProtection(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	t.Setenv("SESSION_KEY", "csrf-login-session-key-32-bytes!!")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}

	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	defer conn.Close()
	hash, err := bcrypt.GenerateFromPassword([]byte("correct horse battery staple"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`CREATE TABLE users(id INTEGER PRIMARY KEY, username TEXT NOT NULL, email TEXT NOT NULL, hashpassword TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`INSERT INTO users(id, username, email, hashpassword) VALUES(1, 'alice', 'alice@example.test', ?)`, string(hash)); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	login.RegisterRoutes(mux, db.New(conn))
	logout.RegisterRoutes(mux)
	protected := httpsecurity.NewCSRFMiddleware([]byte("0123456789abcdef0123456789abcdef"), false)(mux)
	token, csrfCookie := getFormToken(t, protected, "/login")

	withoutToken := postFormRequest("/login", url.Values{"username": {"alice"}, "password": {"correct horse battery staple"}})
	withoutToken.AddCookie(csrfCookie)
	response := httptest.NewRecorder()
	protected.ServeHTTP(response, withoutToken)
	if response.Code != http.StatusForbidden || findCookie(response.Result().Cookies(), "session") != nil {
		t.Fatalf("login without CSRF status=%d session=%v", response.Code, findCookie(response.Result().Cookies(), "session"))
	}

	loginForm := url.Values{
		"username":           {"alice"},
		"password":           {"correct horse battery staple"},
		"gorilla.csrf.Token": {token},
	}
	validLogin := postFormRequest("/login", loginForm)
	validLogin.AddCookie(csrfCookie)
	response = httptest.NewRecorder()
	protected.ServeHTTP(response, validLogin)
	sessionCookie := findCookie(response.Result().Cookies(), "session")
	if response.Code != http.StatusSeeOther || sessionCookie == nil {
		t.Fatalf("valid login status=%d session=%v body=%q", response.Code, sessionCookie, response.Body.String())
	}

	getLogout := httptest.NewRequest(http.MethodGet, "/logout", nil)
	getLogout.AddCookie(sessionCookie)
	response = httptest.NewRecorder()
	protected.ServeHTTP(response, getLogout)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET logout status=%d, want 405", response.Code)
	}

	logoutWithoutToken := postFormRequest("/logout", nil)
	logoutWithoutToken.AddCookie(sessionCookie)
	logoutWithoutToken.AddCookie(csrfCookie)
	response = httptest.NewRecorder()
	protected.ServeHTTP(response, logoutWithoutToken)
	if response.Code != http.StatusForbidden {
		t.Fatalf("logout without CSRF status=%d", response.Code)
	}

	logoutForm := url.Values{"gorilla.csrf.Token": {token}}
	validLogout := postFormRequest("/logout", logoutForm)
	validLogout.AddCookie(sessionCookie)
	validLogout.AddCookie(csrfCookie)
	response = httptest.NewRecorder()
	protected.ServeHTTP(response, validLogout)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("valid logout status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestMarkingAndCRUDPostRoutesAreCoveredGlobally(t *testing.T) {
	t.Setenv("SESSION_KEY", "csrf-route-session-key-32-bytes!!")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	var jobs sync.WaitGroup
	marking.RegisterRoutes(mux, nil, context.Background(), &jobs)
	subjects.RegisterRoutes(mux, nil)
	mux.HandleFunc("GET /csrf-form", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<form method="post">` + csrf.TemplateField(r) + `</form>`))
	})
	protected := httpsecurity.NewCSRFMiddleware([]byte("0123456789abcdef0123456789abcdef"), false)(mux)
	token, cookie := getFormToken(t, protected, "/csrf-form")

	paths := []string{
		"/dashboard/marking/processing",
		"/dashboard/marking/review/apply",
		"/dashboard/marking/artifacts/regenerate",
		"/dashboard/questions/subjects/add",
	}
	for _, path := range paths {
		t.Run(path+" missing token", func(t *testing.T) {
			request := postFormRequest(path, nil)
			request.AddCookie(cookie)
			response := httptest.NewRecorder()
			protected.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
			}
		})
	}

	for _, path := range []string{
		"/dashboard/marking/review/apply",
		"/dashboard/marking/artifacts/regenerate",
		"/dashboard/questions/subjects/add",
	} {
		t.Run(path+" valid token", func(t *testing.T) {
			request := postFormRequest(path, url.Values{"gorilla.csrf.Token": {token}})
			request.AddCookie(cookie)
			response := httptest.NewRecorder()
			protected.ServeHTTP(response, request)
			if response.Code == http.StatusForbidden {
				t.Fatalf("valid CSRF token rejected: %q", response.Body.String())
			}
		})
	}

	// A valid multipart token passes the global CSRF boundary and reaches the
	// existing authentication boundary (which redirects this anonymous test).
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("gorilla.csrf.Token", token); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("pdffile", "copies.pdf")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("%PDF-test"))
	writer.Close()
	request := httptest.NewRequest(http.MethodPost, "/dashboard/marking/processing", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	protected.ServeHTTP(response, request)
	if response.Code == http.StatusForbidden {
		t.Fatalf("valid multipart CSRF token rejected: %q", response.Body.String())
	}
}

func getFormToken(t *testing.T, handler http.Handler, path string) (string, *http.Cookie) {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	matches := integrationTokenPattern.FindStringSubmatch(response.Body.String())
	if len(matches) != 2 {
		t.Fatalf("token missing from %s: status=%d body=%q", path, response.Code, response.Body.String())
	}
	cookie := findCookie(response.Result().Cookies(), "_lazymarking_csrf")
	if cookie == nil {
		t.Fatal("CSRF cookie missing")
	}
	return matches[1], cookie
}

func postFormRequest(path string, values url.Values) *http.Request {
	if values == nil {
		values = url.Values{}
	}
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
