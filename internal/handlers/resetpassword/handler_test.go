package resetpassword

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestResetPasswordHandlerRejectsInvalidPasswordBeforeDatabase(t *testing.T) {
	for _, tc := range []struct {
		name     string
		password string
	}{
		{name: "too short", password: strings.Repeat("a", 11)},
		{name: "too long", password: strings.Repeat("a", 73)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			form := url.Values{
				"token":        {"synthetic-token"},
				"new_password": {tc.password},
			}
			request := httptest.NewRequest(http.MethodPost, "/reset", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()

			// Nil database arguments deliberately prove validation returns before
			// BeginTx, token lookup, and bcrypt processing.
			ResetPasswordHandler(response, request, nil, nil)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
			}
		})
	}
}
