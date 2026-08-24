package login

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"golang.org/x/crypto/bcrypt"
)

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

	if errors.Is(err, sql.ErrNoRows) || bcrypt.CompareHashAndPassword([]byte(userDB.Hashpassword), []byte(password)) != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	session, err := GetStore().Get(r, "session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	// Stocke l’ID et le username de l’utilisateur dans la session
	session.Values["user_id"] = userDB.ID
	session.Values["username"] = userDB.Username

	// Enregistre la session (envoie le cookie au client)
	if err := session.Save(r, w); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Redirige vers /dashboard ou autre
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
