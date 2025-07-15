package login

import (
	"context"
	"net/http"
)

type contextKey string

const (
	userIDKey   contextKey = "user_id"
	usernameKey contextKey = "username"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		userID, ok := session.Values["user_id"].(int64)
		if !ok || userID == 0 {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		username, ok := session.Values["username"].(string)
		if !ok || username == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
		}

		// Ajoute l’ID dans le contexte
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, usernameKey, username)
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
	return id, ok
}

func GetUsername(r *http.Request) (string, bool) {
	username, ok := r.Context().Value(usernameKey).(string)
	return username, ok
}

func FromContext(r *http.Request) (userID int64, username string, ok bool) {
	uidVal := r.Context().Value(userIDKey)
	unameVal := r.Context().Value(usernameKey)

	uid, ok1 := uidVal.(int64)
	uname, ok2 := unameVal.(string)

	return uid, uname, ok1 && ok2
}
