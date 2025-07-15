package login

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultHomeRoutes
	mux.HandleFunc(routes.LoginURL, LoginHandler)
	mux.HandleFunc("POST "+routes.LoginURL, func(w http.ResponseWriter, r *http.Request) {
		LoggedHandler(w, r, queries)
	})
}
