package about

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc(data.DefaultHomeRoutes.AboutURL, AboutHandler)
	mux.HandleFunc("GET /static/about/lazymarking-overview-posca.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "internal/static/about/lazymarking-overview-posca.png")
	})
}
