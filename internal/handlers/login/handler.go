package login

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"golang.org/x/crypto/bcrypt"
)

// A fixed valid bcrypt hash keeps unknown-user and bad-password paths on the
// same password-comparison code path without revealing account existence.
var dummyPasswordHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

var saveLoginSession = func(session *sessions.Session, r *http.Request, w http.ResponseWriter) error {
	return session.Save(r, w)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data := data.HomePageData{
		Routes:    data.DefaultHomeRoutes,
		PageTitle: "Login",
	}

	RenderLoginPage(w, data)
}

func LoggedHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "all field have to be completed", http.StatusBadRequest)
		return
	}

	userDB, err := queries.GetUserByUsername(r.Context(), username)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Error with db", http.StatusInternalServerError)
		return
	}

	hash := dummyPasswordHash
	if err == nil {
		hash = []byte(userDB.Hashpassword)
	}
	passwordErr := bcrypt.CompareHashAndPassword(hash, []byte(password))
	if errors.Is(err, sql.ErrNoRows) || passwordErr != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	if userDB.ID <= 0 || !sessionUsernamePattern.MatchString(userDB.Username) {
		http.Error(w, "Invalid account state", http.StatusInternalServerError)
		return
	}

	if GetStore() == nil {
		http.Error(w, "Session store unavailable", http.StatusInternalServerError)
		return
	}
	session, err := GetStore().Get(r, "session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	// Do not carry arbitrary state from a previous signed identity into the new
	// authenticated cookie.
	session.Values = make(map[interface{}]interface{})
	session.Values["user_id"] = userDB.ID
	session.Values["username"] = userDB.Username

	// Enregistre la session (envoie le cookie au client)
	if err := saveLoginSession(session, r, w); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Redirige vers /dashboard ou autre
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
