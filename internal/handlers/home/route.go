package home

import "net/http"

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", HomeHandler)
}
