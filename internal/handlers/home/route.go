package home

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc(data.DefaultHomeRoutes.Home, HomeHandler)
}
