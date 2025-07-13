package login

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	mux.HandleFunc("/login", LoginHandler)
	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		LoggedHandler(w, r, queries)
	})
}
