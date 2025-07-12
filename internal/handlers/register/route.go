package register

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	mux.HandleFunc("/register", RegisterHandler)
	mux.HandleFunc("POST /register", func(w http.ResponseWriter, r *http.Request) {
		SaveRegisterHandler(w, r, queries)
	})
	mux.HandleFunc("/register/success", RegisterSuccessHandler)
}
