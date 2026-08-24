package login

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInitSessionStoreRequiresStrongKey(t *testing.T) {
	t.Setenv("SESSION_KEY", "too-short")
	t.Setenv("SESSION_SECURE", "")

	if err := InitSessionStore(); err == nil {
		t.Fatal("expected a short session key to be rejected")
	}
}

func TestInitSessionStoreConfiguresSecureCookie(t *testing.T) {
	t.Setenv("SESSION_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("SESSION_SECURE", "true")

	if err := InitSessionStore(); err != nil {
		t.Fatalf("InitSessionStore() error = %v", err)
	}
	if !store.Options.Secure || !store.Options.HttpOnly {
		t.Fatalf("unsafe cookie options: %+v", store.Options)
	}
	if store.Options.SameSite != http.SameSiteStrictMode {
		t.Fatalf("SameSite = %v, want Strict", store.Options.SameSite)
	}
}

func TestAuthMiddlewareStopsAfterIncompleteSession(t *testing.T) {
	t.Setenv("SESSION_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("SESSION_SECURE", "false")
	if err := InitSessionStore(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	recorder := httptest.NewRecorder()
	session, err := store.Get(req, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["user_id"] = int64(1)
	responseWithCookie := httptest.NewRecorder()
	if err := session.Save(req, responseWithCookie); err != nil {
		t.Fatal(err)
	}
	req.AddCookie(responseWithCookie.Result().Cookies()[0])

	called := false
	handler := AuthMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	handler.ServeHTTP(recorder, req)

	if called {
		t.Fatal("protected handler was called with an incomplete session")
	}
	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusFound)
	}
}
