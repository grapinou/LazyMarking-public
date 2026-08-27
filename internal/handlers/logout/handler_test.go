package logout

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grapinou/LazyMarking/internal/handlers/login"
)

func TestLogoutExpiresAndClearsAuthenticatedSession(t *testing.T) {
	t.Setenv("SESSION_KEY", "logout-test-key-32-bytes-long-value")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	seedRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	session, err := login.GetStore().Get(seedRequest, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["user_id"] = int64(1)
	session.Values["username"] = "alice"
	loginResponse := httptest.NewRecorder()
	if err := session.Save(seedRequest, loginResponse); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/logout", nil)
	request.AddCookie(loginResponse.Result().Cookies()[0])

	response := httptest.NewRecorder()
	login.CheckAuth(http.HandlerFunc(LogoutHandler)).ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want 303", response.Code)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge >= 0 {
		t.Fatalf("logout cookie=%v", cookies)
	}

	after := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	after.AddCookie(cookies[0])
	called := false
	afterResponse := httptest.NewRecorder()
	login.CheckAuth(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })).ServeHTTP(afterResponse, after)
	if called || afterResponse.Code != http.StatusFound {
		t.Fatalf("called=%v status=%d, want false/302", called, afterResponse.Code)
	}
}
