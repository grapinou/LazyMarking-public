package register

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Cette méthode de requête n’est pas autorisée.", http.StatusMethodNotAllowed)
		return
	}

	data := data.HomePageData{
		Routes:    data.DefaultHomeRoutes,
		PageTitle: "Créer un compte",
	}

	RenderRegisterPage(w, data)
}

func SaveRegisterHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Cette méthode de requête n’est pas autorisée.", http.StatusMethodNotAllowed)
		return
	}

	// Retrieve data from form
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if username == "" || email == "" || password == "" {
		renderRegistrationError(w, username, email, "Veuillez renseigner tous les champs.", http.StatusBadRequest)
		return
	}
	if err := validateRegistration(username, email, password); err != nil {
		renderRegistrationError(w, username, email, err.Error(), http.StatusBadRequest)
		return
	}

	// hashing password

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		renderRegistrationError(w, username, email, "Impossible d’enregistrer le mot de passe.", http.StatusInternalServerError)
		return
	}

	// save into db

	err = queries.CreateUser(r.Context(), db.CreateUserParams{
		Username:     username,
		Email:        email,
		Hashpassword: string(hashedPassword),
	})
	if err != nil {
		renderRegistrationError(w, username, email, "Impossible de créer ce compte.", http.StatusConflict)
		return
	}

	http.Redirect(w, r, data.DefaultHomeRoutes.RegisterSuccessURL, http.StatusSeeOther)
}

func RegisterSuccessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Cette méthode de requête n’est pas autorisée.", http.StatusMethodNotAllowed)
		return
	}

	data := data.HomePageData{
		Routes:    data.DefaultHomeRoutes,
		PageTitle: "Compte créé",
	}

	RenderSucessRegister(w, data)
}

func renderRegistrationError(w http.ResponseWriter, username, email, message string, status int) {
	RenderRegisterPage(w, data.HomePageData{Routes: data.DefaultHomeRoutes, PageTitle: "Créer un compte", RegisterUsername: username, RegisterEmail: email, RegisterError: message, RegisterStatus: status})
}
