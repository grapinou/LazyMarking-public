package resetpassword

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/grapinou/LazyMarking/internal/db"
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

func TestResetPasswordDoesNotImplicitlyAuthenticate(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`
CREATE TABLE users(id INTEGER PRIMARY KEY,username TEXT,email TEXT,hashpassword TEXT);
CREATE TABLE password_resets(id INTEGER PRIMARY KEY,user_id INTEGER,token TEXT UNIQUE,expires_at DATETIME,used BOOLEAN DEFAULT FALSE);
INSERT INTO users VALUES(1,'alice','alice@example.test','old-hash');`,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec("INSERT INTO password_resets(user_id,token,expires_at) VALUES(1,'valid-token',?)", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	form := url.Values{"token": {"valid-token"}, "new_password": {"new-password-123"}}
	request := httptest.NewRequest(http.MethodPost, "/resetpassword", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	ResetPasswordHandler(response, request, conn, db.New(conn))
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	if len(response.Result().Cookies()) != 0 {
		t.Fatal("password reset emitted an authentication cookie")
	}
	var used bool
	if err := conn.QueryRow("SELECT used FROM password_resets WHERE token='valid-token'").Scan(&used); err != nil || !used {
		t.Fatalf("token used=%v err=%v", used, err)
	}
}

func TestResetEmailDistinguishesUnknownAccountFromDatabaseFailure(t *testing.T) {
	requestFor := func(email string) *http.Request {
		form := url.Values{"email": {email}}
		request := httptest.NewRequest(http.MethodPost, "/sendemailresetpassword", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return request
	}

	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec("CREATE TABLE users(id INTEGER PRIMARY KEY,username TEXT,email TEXT,hashpassword TEXT)"); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	SendResetEmailHandler(response, requestFor("missing@example.test"), db.New(conn))
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/" {
		t.Fatalf("unknown account status=%d location=%q", response.Code, response.Header().Get("Location"))
	}

	if _, err := conn.Exec("DROP TABLE users"); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	SendResetEmailHandler(response, requestFor("any@example.test"), db.New(conn))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("database failure status=%d, want 500", response.Code)
	}
}
