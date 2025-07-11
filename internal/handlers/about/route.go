package about

import "net/http"

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/about", AboutHandler)
}