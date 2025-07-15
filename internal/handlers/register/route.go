package register

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultHomeRoutes
	mux.HandleFunc(routes.RegisterURL, RegisterHandler)
	mux.HandleFunc("POST "+routes.RegisterURL, func(w http.ResponseWriter, r *http.Request) {
		SaveRegisterHandler(w, r, queries)
	})
	mux.HandleFunc(routes.RegisterSuccessURL, RegisterSuccessHandler)
}
