package register

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := data.TemplateLayoutHomeData{
		Home:              "/",
		AboutURL:          "/about",
		LoginURL:          "/login",
		RegisterURL:       "/register",
		ForgotPasswordURL: "/forgot-password",
		PageTitle:         "Register",
	}

	RenderRegisterPage(w, data)
}

func SaveRegisterHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusInternalServerError)
	}

	// Retrieve data from form
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if username == "" || email == "" || password == "" {
		http.Error(w, "All field have to be completed", http.StatusBadRequest)
		return
	}

	// hashing password

	hasshedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Hasshing process failed : "+err.Error(), http.StatusInternalServerError)
		return
	}

	// save into db

	err = queries.CreateUser(r.Context(), db.CreateUserParams{
		Username:     username,
		Email:        email,
		Hashpassword: string(hasshedPassword),
	})
	if err != nil {
		http.Error(w, "Error to registration into db : fields need to be unique. "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/register/success", http.StatusSeeOther)
}

func RegisterSuccessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := data.TemplateLayoutHomeData{
		Home:              "/",
		AboutURL:          "/about",
		LoginURL:          "/login",
		RegisterURL:       "/register",
		ForgotPasswordURL: "/forgot-password",
		PageTitle:         "Success",
	}

	RenderSucessRegister(w, data)
}
