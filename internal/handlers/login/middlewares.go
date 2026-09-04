package login

import (
	"context"
	"net/http"
	"regexp"
)

type contextKey string

const (
	userIDKey   contextKey = "user_id"
	usernameKey contextKey = "username"
)

var sessionUsernamePattern = regexp.MustCompile(`^[[:alnum:]_.-]{3,64}$`)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			http.Error(w, "Session store unavailable", http.StatusInternalServerError)
			return
		}

		session, err := GetSession(r)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		userID, ok := session.Values["user_id"].(int64)
		if !ok || userID <= 0 {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		username, ok := session.Values["username"].(string)
		if !ok || !sessionUsernamePattern.MatchString(username) {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Ajoute l’ID dans le contexte
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, usernameKey, username)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		username, ok := GetUsername(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, usernameKey, username)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(r *http.Request) (int64, bool) {
	id, ok := r.Context().Value(userIDKey).(int64)
	return id, ok && id > 0
}

func GetUsername(r *http.Request) (string, bool) {
	username, ok := r.Context().Value(usernameKey).(string)
	return username, ok && sessionUsernamePattern.MatchString(username)
}

func FromContext(r *http.Request) (userID int64, username string, ok bool) {
	uidVal := r.Context().Value(userIDKey)
	unameVal := r.Context().Value(usernameKey)

	uid, ok1 := uidVal.(int64)
	uname, ok2 := unameVal.(string)

	return uid, uname, ok1 && ok2 && uid > 0 && sessionUsernamePattern.MatchString(uname)
}

// CheckAuth enchaîne ContextMiddleware et AuthMiddleware autour d'un http.Handler
func CheckAuth(handler http.Handler) http.Handler {
	return AuthMiddleware(
		ContextMiddleware(handler),
	)
}
