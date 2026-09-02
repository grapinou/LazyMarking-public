package httpsecurity

import (
	"bytes"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
)

var csrfTokenPattern = regexp.MustCompile(`name="gorilla\.csrf\.Token" value="([^"]+)"`)

func TestCSRFMiddlewareSafeAndUnsafeMethods(t *testing.T) {
	var calls atomic.Int64
	handler := NewCSRFMiddleware([]byte("0123456789abcdef0123456789abcdef"), false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method == http.MethodGet {
			tmpl := template.Must(template.New("form").Funcs(TemplateFuncs(w)).Parse(`<form method="post">{{ csrfField }}</form>`))
			if err := tmpl.Execute(w, nil); err != nil {
				t.Fatal(err)
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	token, cookie := issueCSRFToken(t, handler)
	if calls.Load() != 1 {
		t.Fatalf("GET calls=%d", calls.Load())
	}
	if !cookie.HttpOnly || cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" {
		t.Fatalf("CSRF cookie options=%+v", cookie)
	}

	for _, tc := range []struct {
		name   string
		token  string
		cookie *http.Cookie
		want   int
	}{
		{name: "missing", cookie: cookie, want: http.StatusForbidden},
		{name: "invalid", token: "invalid", cookie: cookie, want: http.StatusForbidden},
		{name: "valid", token: token, cookie: cookie, want: http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			form := url.Values{}
			if tc.token != "" {
				form.Set("gorilla.csrf.Token", tc.token)
			}
			request := httptest.NewRequest(http.MethodPost, "/mutate", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.AddCookie(tc.cookie)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.want {
				t.Fatalf("status=%d body=%q, want %d", response.Code, response.Body.String(), tc.want)
			}
			if tc.want == http.StatusForbidden && !strings.Contains(response.Body.String(), "Rechargez la page") {
				t.Fatalf("unsafe CSRF error=%q", response.Body.String())
			}
		})
	}
	if calls.Load() != 2 {
		t.Fatalf("handler calls=%d, rejected POST reached mutation", calls.Load())
	}
}

func TestCSRFMiddlewareRejectsTokenFromAnotherCookie(t *testing.T) {
	handler := NewCSRFMiddleware([]byte("0123456789abcdef0123456789abcdef"), false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		tmpl := template.Must(template.New("form").Funcs(TemplateFuncs(w)).Parse(`{{ csrfField }}`))
		_ = tmpl.Execute(w, nil)
	}))
	tokenA, _ := issueCSRFToken(t, handler)
	_, cookieB := issueCSRFToken(t, handler)

	form := url.Values{"gorilla.csrf.Token": {tokenA}}
	request := httptest.NewRequest(http.MethodPost, "/mutate", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(cookieB)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", response.Code)
	}
}

func TestCSRFMiddlewareAcceptsMultipartForm(t *testing.T) {
	reached := false
	handler := NewCSRFMiddleware([]byte("0123456789abcdef0123456789abcdef"), false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusNoContent)
	}))
	// Use a rendering endpoint to obtain the cookie and masked form token.
	renderer := NewCSRFMiddleware([]byte("0123456789abcdef0123456789abcdef"), false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		tmpl := template.Must(template.New("form").Funcs(TemplateFuncs(w)).Parse(`{{ csrfField }}`))
		_ = tmpl.Execute(w, nil)
	}))
	token, cookie := issueCSRFToken(t, renderer)

	// The cookie is authenticated by the same key and is accepted by the
	// independently constructed process-global middleware.
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("gorilla.csrf.Token", token); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("pdffile", "copies.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("%PDF-test")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || !reached {
		t.Fatalf("status=%d reached=%v body=%q", response.Code, reached, response.Body.String())
	}
}

func TestCSRFConfigurationRequiresDedicatedKey(t *testing.T) {
	t.Setenv("SESSION_SECURE", "false")
	t.Setenv("CSRF_AUTH_KEY", "")
	if _, err := NewCSRFMiddlewareFromEnvironment(); err == nil {
		t.Fatal("missing CSRF_AUTH_KEY accepted")
	}
	t.Setenv("CSRF_AUTH_KEY", "0123456789abcdef0123456789abcdef")
	if _, err := NewCSRFMiddlewareFromEnvironment(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SESSION_KEY", "0123456789abcdef0123456789abcdef")
	if _, err := NewCSRFMiddlewareFromEnvironment(); err == nil {
		t.Fatal("CSRF_AUTH_KEY identical to SESSION_KEY accepted")
	}
}

func TestAllUnsafeHTMLFormsContainCentralCSRFField(t *testing.T) {
	root := filepath.Join("..", "templates")
	formPattern := regexp.MustCompile(`(?is)<form\b[^>]*\bmethod\s*=\s*"(?:post|put|patch|delete)"[^>]*>(.*?)</form>`)
	forms := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range formPattern.FindAllSubmatch(content, -1) {
			forms++
			if !bytes.Contains(match[1], []byte("{{ csrfField }}")) {
				t.Errorf("unsafe form without csrfField: %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if forms != 73 {
		t.Fatalf("unsafe form inventory=%d, want 73", forms)
	}
}

func issueCSRFToken(t *testing.T, handler http.Handler) (string, *http.Cookie) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/form", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%q", response.Code, response.Body.String())
	}
	matches := csrfTokenPattern.FindStringSubmatch(response.Body.String())
	if len(matches) != 2 || matches[1] == "" {
		t.Fatalf("CSRF token absent from form: %q", response.Body.String())
	}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "_lazymarking_csrf" {
			return matches[1], cookie
		}
	}
	t.Fatal("CSRF cookie absent")
	return "", nil
}

func TestCSRFErrorDoesNotExposeFailureReason(t *testing.T) {
	handler := NewCSRFMiddleware([]byte("0123456789abcdef0123456789abcdef"), false)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler reached")
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/resource", nil))
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d", response.Code)
	}
	for _, forbidden := range []string{"gorilla", "token", "CSRF"} {
		if strings.Contains(response.Body.String(), forbidden) {
			t.Fatalf("technical detail %q exposed in %q", forbidden, response.Body.String())
		}
	}
}
