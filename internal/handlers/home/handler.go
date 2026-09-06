package home

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Cette méthode de requête n’est pas autorisée.", http.StatusMethodNotAllowed)
		return
	}

	data := data.HomePageData{
		Routes:    data.DefaultHomeRoutes,
		PageTitle: "Accueil",
	}

	RenderHomePage(w, data)
}
