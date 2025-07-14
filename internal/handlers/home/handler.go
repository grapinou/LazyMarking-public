package home

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := data.HomePageData{
		Routes:    data.DefaultHomeRoutes,
		PageTitle: "Home",
	}

	RenderHomePage(w, data)
}
