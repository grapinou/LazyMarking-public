package marking

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/httpsecurity"
)

func TestStudentCopyHTTPSCouponAndPublicationCSRF(t *testing.T) {
	t.Setenv("SESSION_KEY", "p4-test-session-key-with-more-than-32-characters")
	t.Setenv("SESSION_SECURE", "true")
	t.Setenv("APP_BASE_URL", "https://school.example")
	f, mux := newCumulativeFixture(t)
	if err := ensureStudentAccess(t.Context(), f.queries, 1, 10); err != nil {
		t.Fatal(err)
	}
	rows, err := f.queries.ListStudentCopyAccess(t.Context(), 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	var token, otherToken string
	for _, row := range rows {
		if row.StudentExamID == 100 {
			token = row.Token
		}
		if row.StudentExamID == 106 {
			otherToken = row.Token
		}
	}
	if token == "" || otherToken == "" {
		t.Fatal("synthetic coupons missing")
	}
	handler := tools.SecurityHeaders(httpsecurity.NewCSRFMiddleware(
		[]byte("0123456789abcdef0123456789abcdef"), true, "school.example",
	)(mux))
	request := func(method, path string, form url.Values, referer string, teacher bool, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "http://backend.internal:8080"+path, strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Referer", referer)
		for _, cookie := range cookies {
			if cookie != nil {
				r.AddCookie(cookie)
			}
		}
		if teacher {
			r.AddCookie(markingSessionCookie(t, r))
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		return rec
	}
	assertCopyHeaders := func(rec *httptest.ResponseRecorder) {
		t.Helper()
		if got := rec.Header().Get("Referrer-Policy"); got != "origin" {
			t.Fatalf("copy Referrer-Policy=%q", got)
		}
		if got := rec.Header().Get("Cache-Control"); got != "no-store, private" {
			t.Fatalf("copy Cache-Control=%q", got)
		}
	}
	findCookie := func(rec *httptest.ResponseRecorder, name string) *http.Cookie {
		t.Helper()
		for _, cookie := range rec.Result().Cookies() {
			if cookie.Name == name {
				return cookie
			}
		}
		t.Fatalf("missing cookie %s", name)
		return nil
	}
	path := "/copies/" + token
	page := request("GET", path, nil, "", false)
	assertCopyHeaders(page)
	if page.Code != 200 || strings.Contains(page.Body.String(), "Alice") {
		t.Fatal("coupon challenge failed or disclosed student identity")
	}
	match := regexp.MustCompile(`name="gorilla\.csrf\.Token" value="([^"]+)"`).FindStringSubmatch(page.Body.String())
	if len(match) != 2 {
		t.Fatal("coupon CSRF field missing")
	}
	// Read the field as a browser does: html/template escapes '+' in attributes.
	csrfToken := html.UnescapeString(match[1])
	csrfCookie := findCookie(page, "_lazymarking_csrf")
	for _, tc := range []struct {
		name, csrfToken, referer string
	}{
		{"missing token", "", "https://school.example/"},
		{"invalid token", "invalid", "https://school.example/"},
		{"foreign referer", csrfToken, "https://evil.example/"},
		{"missing referer", csrfToken, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := request("POST", path, url.Values{"code": {couponCode(token)}, "gorilla.csrf.Token": {tc.csrfToken}}, tc.referer, false, csrfCookie)
			if got.Code != 403 {
				t.Fatalf("forbidden coupon request status=%d", got.Code)
			}
			for _, cookie := range got.Result().Cookies() {
				if cookie.Name == studentCopyCookie {
					t.Fatal("forbidden coupon request established student session")
				}
			}
		})
	}
	form := url.Values{"code": {couponCode(token)}, "gorilla.csrf.Token": {csrfToken}}
	authenticated := request("POST", path, form, "https://school.example/", false, csrfCookie)
	assertCopyHeaders(authenticated)
	if authenticated.Code != 200 || !strings.Contains(authenticated.Body.String(), "pas encore disponible") {
		t.Fatalf("HTTPS coupon status=%d", authenticated.Code)
	}
	studentCookie := findCookie(authenticated, studentCopyCookie)
	if !studentCookie.Secure || !studentCookie.HttpOnly || studentCookie.SameSite != http.SameSiteLaxMode || studentCookie.Path != "/copies/" {
		t.Fatalf("student cookie options=%+v", studentCookie)
	}
	publicationPath := "/dashboard/marking/student-access"
	publication := url.Values{"action": {"publish"}, "exam_generated_id": {"10"}, "gorilla.csrf.Token": {csrfToken}}
	if got := request("POST", publicationPath, publication, "https://school.example/", false, csrfCookie, studentCookie); got.Code != http.StatusFound || got.Header().Get("Location") != "/login" {
		t.Fatalf("student session accepted as teacher: status=%d location=%q", got.Code, got.Header().Get("Location"))
	}
	withoutToken := url.Values{"action": {"publish"}, "exam_generated_id": {"10"}}
	if got := request("POST", publicationPath, withoutToken, "https://school.example/", true, csrfCookie); got.Code != 403 {
		t.Fatalf("teacher publication without CSRF status=%d", got.Code)
	}
	access, err := f.queries.GetStudentCopyAccessByToken(t.Context(), token)
	if err != nil || access.State != "unpublished" || access.FailedAttempts != 0 {
		t.Fatal("rejected requests changed copy access")
	}
	published := request("POST", publicationPath, publication, "https://school.example/", true, csrfCookie)
	if published.Code != http.StatusSeeOther || published.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Fatalf("HTTPS teacher publication status=%d", published.Code)
	}
	access, err = f.queries.GetStudentCopyAccessByToken(t.Context(), token)
	if err != nil || access.State != "published" {
		t.Fatal("HTTPS teacher publication did not persist")
	}
	available := request("GET", path, nil, "", false, studentCookie)
	assertCopyHeaders(available)
	if available.Code != 200 || !strings.Contains(available.Body.String(), path+"/pdf") {
		t.Fatal("published copy not available through coupon session")
	}
	for _, denied := range []string{"/copies/unknown", path + "/pdf", "/copies/" + otherToken + "/pdf"} {
		got := request("GET", denied, nil, "", false)
		if denied == "/copies/"+otherToken+"/pdf" {
			got = request("GET", denied, nil, "", false, studentCookie)
		}
		assertCopyHeaders(got)
		if got.Code != 404 {
			t.Fatalf("forbidden copy status=%d", got.Code)
		}
	}
}
