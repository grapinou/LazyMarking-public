package login

import (
	"errors"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/gorilla/sessions"
)

var store *sessions.CookieStore
var sessionCookieName string

const defaultSessionCookieName = "session"

var sessionCookieNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func InitSessionStore() error {
	key := os.Getenv("SESSION_KEY")
	if len(key) < 32 {
		return errors.New("SESSION_KEY must contain at least 32 characters")
	}
	secureValue := os.Getenv("SESSION_SECURE")
	if secureValue == "" {
		return errors.New("SESSION_SECURE must be explicitly set to true or false")
	}
	secure, err := strconv.ParseBool(secureValue)
	if err != nil {
		return errors.New("SESSION_SECURE must be a boolean")
	}
	cookieName := os.Getenv("SESSION_COOKIE_NAME")
	if cookieName == "" {
		cookieName = defaultSessionCookieName
	}
	if !sessionCookieNamePattern.MatchString(cookieName) {
		return errors.New("SESSION_COOKIE_NAME contains invalid characters")
	}

	store = sessions.NewCookieStore([]byte(key))
	sessionCookieName = cookieName
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   7200,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
	return nil
}

func GetSession(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, sessionCookieName)
}

func NewSession() *sessions.Session {
	session := sessions.NewSession(store, sessionCookieName)
	options := *store.Options
	session.Options = &options
	return session
}

func GetStore() *sessions.CookieStore {
	return store
}
