package resetpassword

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/mailer"
	"golang.org/x/crypto/bcrypt"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func ShowRequestFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed from ShowRequestResetFormHandler", http.StatusMethodNotAllowed)
		return
	}

	data := data.HomePageData{
		Routes:    data.DefaultHomeRoutes,
		PageTitle: "Request-Reset-Password",
	}

	RenderShowRequestForm(w, data)
}

func SendResetEmailHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed from SendResetEmailHandler", http.StatusMethodNotAllowed)
		return
	}

	routes := data.DefaultHomeRoutes

	userEmail := r.FormValue("email")
	userDB, err := queries.GetUserByEmail(r.Context(), userEmail)

	if err != nil || userDB.Email == "" {
		// Par sécurité, on ne révèle pas si l'email existe ou non
		http.Redirect(w, r, routes.Home, http.StatusSeeOther)
		return
	}

	token := uuid.NewString()
	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	err = queries.CreateResetPassword(r.Context(), db.CreateResetPasswordParams{
		UserID:    userDB.ID,
		Token:     token,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		http.Error(w, "Can't create reset password link", http.StatusInternalServerError)
		return
	}

	resetLink := fmt.Sprintf("%s%s?token=%s", config.GetBaseURL(), routes.FormResetPasswordURL, token)
	err = mailer.SendResetEmail(userDB.Username, userDB.Email, resetLink)
	if err != nil {
		http.Error(w, "Can't send email", http.StatusInternalServerError)
		return
	}
	log.Println("Email send to : ", userDB.Email)

	http.Redirect(w, r, routes.Home, http.StatusSeeOther)
}

func ShowResetFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed from ShowResetFormHandler", http.StatusMethodNotAllowed)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token missing", http.StatusBadRequest)
		return
	}

	data := data.HomePageData{
		Routes:    data.DefaultHomeRoutes,
		PageTitle: "Form-Reset-Password",
		ExtraData: map[string]any{
			"Token": token,
		},
	}

	RenderShowResetForm(w, data)
}

func ResetPasswordHandler(w http.ResponseWriter, r *http.Request, conn *sql.DB, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed from ResetPasswordHandler", http.StatusMethodNotAllowed)
		return
	}

	token := r.FormValue("token")
	newPassword := r.FormValue("new_password")

	if token == "" {
		http.Error(w, "No token", http.StatusBadRequest)
		return
	}
	if err := tools.ValidatePassword(newPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Démarre une transaction
	tx, err := conn.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() // rollback automatique en cas d'erreur

	qtx := queries.WithTx(tx)

	resetValidation, err := qtx.GetResetPasswordByToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Invalid token or link expire", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Hashing process failed", http.StatusInternalServerError)
		return
	}

	rows, err := qtx.UpdateUserPassword(r.Context(), db.UpdateUserPasswordParams{
		Hashpassword: string(hashedPassword),
		ID:           resetValidation.UserID,
	})
	if err != nil {
		http.Error(w, "userpassword not update", http.StatusInternalServerError)
		return
	}
	if rows != 1 {
		log.Printf("From ResetPasswordHandler -> UpdateUserPassword affected %d rows for user %d", rows, resetValidation.UserID)
		http.Error(w, "userpassword not update", http.StatusInternalServerError)
		return
	}

	// Invalide tous les tokens de reset de l'utilisateur.
	err = qtx.MarkAllResetPasswordTokensUsedForUser(r.Context(), resetValidation.UserID)
	if err != nil {
		http.Error(w, "Failed to invalidate reset tokens", http.StatusInternalServerError)
		return
	}

	// Valide la transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultHomeRoutes.LoginURL+"?reset=success", http.StatusSeeOther)
}
