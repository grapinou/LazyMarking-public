package login

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

var store *sessions.CookieStore

func InitSessionStore() {
	key := os.Getenv("SESSION_KEY")
	if key == "" {
		log.Println("SESSION_KEY missing: using fallback dev key")
		key = "dev-session-key-change-me"
	} else {
		log.Println("SESSION_KEY loaded")
	}

	store = sessions.NewCookieStore([]byte(key))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   7200,
		HttpOnly: true,
		Secure:   false, // Mettre à true avec HTTPS
		SameSite: http.SameSiteStrictMode,
	}
}

func GetStore() *sessions.CookieStore {
	return store
}
