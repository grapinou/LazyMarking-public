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
		http.Error(w, "Cette méthode de requête n’est pas autorisée.", http.StatusMethodNotAllowed)
		return
	}
	data := data.HomePageData{
		Routes:    data.DefaultHomeRoutes,
		PageTitle: "Connexion",
	}

	RenderLoginPage(w, data)
}

func LoggedHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Cette méthode de requête n’est pas autorisée.", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "Veuillez renseigner tous les champs.", http.StatusBadRequest)
		return
	}

	userDB, err := queries.GetUserByUsername(r.Context(), username)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Impossible d’accéder aux données demandées.", http.StatusInternalServerError)
		return
	}

	hash := dummyPasswordHash
	if err == nil {
		hash = []byte(userDB.Hashpassword)
	}
	passwordErr := bcrypt.CompareHashAndPassword(hash, []byte(password))
	if errors.Is(err, sql.ErrNoRows) || passwordErr != nil {
		http.Error(w, "Nom d’utilisateur ou mot de passe incorrect.", http.StatusUnauthorized)
		return
	}
	if userDB.ID <= 0 || !sessionUsernamePattern.MatchString(userDB.Username) {
		http.Error(w, "Impossible d’accéder à ce compte.", http.StatusInternalServerError)
		return
	}

	if GetStore() == nil {
		http.Error(w, "La connexion est temporairement indisponible.", http.StatusInternalServerError)
		return
	}
	session, err := GetSession(r)
	if err != nil {
		// A stale, corrupt, or differently signed cookie is unauthenticated
		// client state. Replace it instead of failing a valid login.
		session = NewSession()
	}

	// Do not carry arbitrary state from a previous signed identity into the new
	// authenticated cookie.
	session.Values = make(map[interface{}]interface{})
	session.Values["user_id"] = userDB.ID
	session.Values["username"] = userDB.Username

	// Enregistre la session (envoie le cookie au client)
	if err := saveLoginSession(session, r, w); err != nil {
		http.Error(w, "Impossible d’ouvrir la session.", http.StatusInternalServerError)
		return
	}

	// Redirige vers /dashboard ou autre
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
