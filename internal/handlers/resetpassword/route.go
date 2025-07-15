package resetpassword

import (
	"database/sql"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, conn *sql.DB, queries *db.Queries) {
	routes := data.DefaultHomeRoutes

	mux.HandleFunc(routes.RequestResetPasswordURL, ShowRequestFormHandler)
	mux.HandleFunc("POST "+routes.SendEmailResetPasswordURL, func(w http.ResponseWriter, r *http.Request) {
		SendResetEmailHandler(w, r, queries)
	})

	mux.HandleFunc(routes.FormResetPasswordURL, ShowResetFormHandler)

	mux.HandleFunc(routes.ResetPasswordURL, func(w http.ResponseWriter, r *http.Request) {
		ResetPasswordHandler(w, r, conn, queries)
	})
}
