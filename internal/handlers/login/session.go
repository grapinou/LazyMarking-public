package login

import (
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/sessions"
)

var store *sessions.CookieStore

func InitSessionStore() error {
	key := os.Getenv("SESSION_KEY")
	if len(key) < 32 {
		return errors.New("SESSION_KEY must contain at least 32 characters")
	}
	secure, err := strconv.ParseBool(os.Getenv("SESSION_SECURE"))
	if err != nil && os.Getenv("SESSION_SECURE") != "" {
		return errors.New("SESSION_SECURE must be a boolean")
	}

	store = sessions.NewCookieStore([]byte(key))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   7200,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
	return nil
}

func GetStore() *sessions.CookieStore {
	return store
}
