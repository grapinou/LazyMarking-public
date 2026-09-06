package register

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRegistrationErrorKeepsOnlyNonSensitiveValues(t *testing.T) {
	t.Chdir("../../..")
	for _, values := range []url.Values{
		{"username": {"prof.demo"}, "email": {"prof@example.org"}, "password": {"secret"}},
		{"username": {"prof.demo"}, "email": {"invalid"}, "password": {"secret-long-password"}},
		{"username": {"prof.demo"}, "email": {"prof@example.org"}, "password": {""}},
	} {
		r := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(values.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		SaveRegisterHandler(w, r, nil)
		body := w.Body.String()
		if w.Code != 400 || !strings.Contains(body, `role="alert"`) || !strings.Contains(body, `value="prof.demo"`) || !strings.Contains(body, `value="`+values.Get("email")+`"`) {
			t.Fatalf("unexpected form response: %d %s", w.Code, body)
		}
		if strings.Contains(body, "octets") || (values.Get("password") != "" && strings.Contains(body, values.Get("password"))) {
			t.Fatal("technical message or password leaked")
		}
	}
}
