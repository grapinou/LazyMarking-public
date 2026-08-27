package login

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testSessionKey = "0123456789abcdef0123456789abcdef"

func TestInitSessionStoreRequiresSessionKey(t *testing.T) {
	t.Setenv("SESSION_KEY", "")
	t.Setenv("SESSION_SECURE", "true")
	if err := InitSessionStore(); err == nil {
		t.Fatal("expected a missing session key to be rejected")
	}
}

func TestInitSessionStoreRequiresStrongKey(t *testing.T) {
	t.Setenv("SESSION_KEY", "too-short")
	t.Setenv("SESSION_SECURE", "")

	if err := InitSessionStore(); err == nil {
		t.Fatal("expected a short session key to be rejected")
	}
}

func TestInitSessionStoreConfiguresSecureCookie(t *testing.T) {
	t.Setenv("SESSION_KEY", testSessionKey)
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
	if store.Options.Path != "/" || store.Options.MaxAge != 7200 {
		t.Fatalf("unexpected cookie scope or lifetime: %+v", store.Options)
	}
}

func TestInitSessionStoreRequiresExplicitValidSecureSetting(t *testing.T) {
	for _, value := range []string{"", "sometimes"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("SESSION_KEY", testSessionKey)
			t.Setenv("SESSION_SECURE", value)
			if err := InitSessionStore(); err == nil {
				t.Fatalf("SESSION_SECURE=%q was accepted", value)
			}
		})
	}
}

func TestAuthMiddlewareStopsAfterIncompleteSession(t *testing.T) {
	t.Setenv("SESSION_KEY", testSessionKey)
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

func TestAuthMiddlewareRejectsMissingTamperedAndInvalidSessions(t *testing.T) {
	initTestSessionStore(t)
	tests := []struct {
		name   string
		values map[interface{}]interface{}
		tamper bool
	}{
		{name: "missing cookie"},
		{name: "invalid user type", values: map[interface{}]interface{}{"user_id": "1", "username": "alice"}},
		{name: "negative user", values: map[interface{}]interface{}{"user_id": int64(-1), "username": "alice"}},
		{name: "invalid username", values: map[interface{}]interface{}{"user_id": int64(1), "username": "../alice"}},
		{name: "tampered", values: map[interface{}]interface{}{"user_id": int64(1), "username": "alice"}, tamper: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			seedRequest := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
			req := seedRequest
			if tc.values != nil {
				cookie := saveTestSessionCookie(t, seedRequest, tc.values)
				if tc.tamper {
					index := 0
					replacement := byte('A')
					if cookie.Value[index] == replacement {
						replacement = 'B'
					}
					cookie.Value = cookie.Value[:index] + string(replacement) + cookie.Value[index+1:]
				}
				req = httptest.NewRequest(http.MethodGet, "/dashboard", nil)
				req.AddCookie(cookie)
			}
			called := false
			recorder := httptest.NewRecorder()
			CheckAuth(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })).ServeHTTP(recorder, req)
			if called || recorder.Code != http.StatusFound {
				t.Fatalf("called=%v status=%d, want false/302", called, recorder.Code)
			}
		})
	}
}

func TestCheckAuthPropagatesValidatedIdentityAndDisablesCaching(t *testing.T) {
	initTestSessionStore(t)
	seedRequest := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	cookie := saveTestSessionCookie(t, seedRequest, map[interface{}]interface{}{"user_id": int64(7), "username": "alice"})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(cookie)
	recorder := httptest.NewRecorder()
	called := false
	CheckAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		userID, username, ok := FromContext(r)
		if !ok || userID != 7 || username != "alice" {
			t.Fatalf("identity=(%d,%q,%v)", userID, username, ok)
		}
	})).ServeHTTP(recorder, req)
	if !called || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("called=%v Cache-Control=%q", called, recorder.Header().Get("Cache-Control"))
	}
}

func TestContextAccessorsRejectInvalidInjectedValues(t *testing.T) {
	ctx := context.WithValue(context.Background(), userIDKey, int64(0))
	ctx = context.WithValue(ctx, usernameKey, "../alice")
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	if _, _, ok := FromContext(req); ok {
		t.Fatal("FromContext accepted invalid identity")
	}
}

func initTestSessionStore(t *testing.T) {
	t.Helper()
	t.Setenv("SESSION_KEY", testSessionKey)
	t.Setenv("SESSION_SECURE", "false")
	if err := InitSessionStore(); err != nil {
		t.Fatal(err)
	}
}

func saveTestSessionCookie(t *testing.T, req *http.Request, values map[interface{}]interface{}) *http.Cookie {
	t.Helper()
	session, err := store.Get(req, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values = values
	recorder := httptest.NewRecorder()
	if err := session.Save(req, recorder); err != nil {
		t.Fatal(err)
	}
	return recorder.Result().Cookies()[0]
}
