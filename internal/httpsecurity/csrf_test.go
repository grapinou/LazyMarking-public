package httpsecurity

import (
	"bytes"
	"html/template"
	"log"
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
	// Includes family and QCM sharing/copy forms.
	if forms != 84 {
		t.Fatalf("unsafe form inventory=%d, want 84", forms)
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
	var diagnostics bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&diagnostics)
	t.Cleanup(func() { log.SetOutput(previousOutput) })
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
	if diagnostics.Len() != 0 {
		t.Fatal("CSRF rejection emitted diagnostics that may contain private URLs")
	}
}

func TestCSRFConfigurationPublicURL(t *testing.T) {
	t.Setenv("SESSION_SECURE", "true")
	t.Setenv("SESSION_KEY", "a-distinct-test-session-key-with-32-characters")
	t.Setenv("CSRF_AUTH_KEY", "0123456789abcdef0123456789abcdef")
	for _, tc := range []struct {
		base  string
		valid bool
	}{
		{"", true},
		{"https://school.example:8443", true},
		{"http://localhost:8080", true},
		{"https://school.example/prefix", true},
		{"/relative", false},
		{"//school.example", false},
		{"ftp://school.example", false},
		{"https://user:password@school.example", false},
		{"https://school.example?token=private", false},
		{"https://school.example?", false},
		{"https://school.example#private", false},
		{"https://:8443", false},
		{"https://%", false},
	} {
		t.Run(tc.base, func(t *testing.T) {
			t.Setenv("APP_BASE_URL", tc.base)
			_, err := NewCSRFMiddlewareFromEnvironment()
			if (err == nil) != tc.valid {
				t.Fatalf("configuration valid=%v, want %v: %v", err == nil, tc.valid, err)
			}
		})
	}
}

func TestCSRFHTTPSPublicOriginBehindProxy(t *testing.T) {
	t.Setenv("SESSION_SECURE", "true")
	t.Setenv("SESSION_KEY", "a-distinct-test-session-key-with-32-characters")
	t.Setenv("CSRF_AUTH_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("APP_BASE_URL", "https://school.example:8443")
	middleware, err := NewCSRFMiddlewareFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	var mutations atomic.Int64
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl := template.Must(template.New("form").Funcs(TemplateFuncs(w)).Parse(`{{ csrfField }}`))
			_ = tmpl.Execute(w, nil)
			return
		}
		mutations.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	token, cookie := issueCSRFToken(t, handler)
	_, otherCookie := issueCSRFToken(t, handler)
	safeRequest := httptest.NewRequest(http.MethodGet, "/form", nil)
	safeRequest.Header.Set("Origin", "http://external.example")
	safeResponse := httptest.NewRecorder()
	handler.ServeHTTP(safeResponse, safeRequest)
	if safeResponse.Code != http.StatusOK {
		t.Fatal("safe GET was subjected to unsafe-method origin checks")
	}
	if !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" {
		t.Fatalf("CSRF cookie options=%+v", cookie)
	}
	for _, tc := range []struct {
		name, origin, referer, token string
		cookie                       *http.Cookie
		want                         int
	}{
		{"public origin", "https://school.example:8443", "", token, cookie, 204},
		{"origin-only referer", "", "https://school.example:8443/", token, cookie, 204},
		{"missing token", "https://school.example:8443", "", "", cookie, 403},
		{"invalid token", "https://school.example:8443", "", "invalid", cookie, 403},
		{"missing cookie", "https://school.example:8443", "", token, nil, 403},
		{"other cookie", "https://school.example:8443", "", token, otherCookie, 403},
		{"untrusted origin", "https://evil.example", "https://school.example:8443/", token, cookie, 403},
		{"untrusted referer", "", "https://evil.example/", token, cookie, 403},
		{"missing origin and referer", "", "", token, cookie, 403},
		{"cleartext referer", "", "http://school.example:8443/", token, cookie, 403},
		{"cleartext trusted origin", "http://school.example:8443", "", token, cookie, 403},
		{"non-HTTP trusted origin", "ftp://school.example:8443", "", token, cookie, 403},
		{"origin with credentials", "https://user@school.example:8443", "", token, cookie, 403},
		{"origin with path", "https://school.example:8443/private", "", token, cookie, 403},
		{"wrong port", "https://school.example", "", token, cookie, 403},
		{"subdomain", "https://other.school.example:8443", "", token, cookie, 403},
		{"host suffix", "https://school.example.evil:8443", "", token, cookie, 403},
		{"null origin", "null", "", token, cookie, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			form := url.Values{"gorilla.csrf.Token": {tc.token}}
			r := httptest.NewRequest(http.MethodPost, "http://backend.internal:8080/publish", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("Referer", tc.referer)
			// Spoofing these headers must not expand the configured allowlist.
			r.Header.Set("X-Forwarded-Host", "evil.example")
			r.Header.Set("X-Forwarded-Proto", "https")
			if tc.cookie != nil {
				r.AddCookie(tc.cookie)
			}
			before := mutations.Load()
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, r)
			if response.Code != tc.want {
				t.Fatalf("status=%d, want %d", response.Code, tc.want)
			}
			if tc.want == 403 && mutations.Load() != before {
				t.Fatal("forbidden request reached mutation handler")
			}
		})
	}
}
