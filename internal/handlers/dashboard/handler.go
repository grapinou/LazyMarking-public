package dashboard

import (
	"log"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Cette méthode de requête n’est pas autorisée.", http.StatusMethodNotAllowed)
		return
	}

	userID, username, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Authentification requise.", http.StatusUnauthorized)
		return
	}

	log.Println("Logged as : ", userID)
	log.Println("username : ", username)

	data := data.DashboardPageData{
		Routes:    data.DefaultDashboardRoutes,
		PageTitle: "Tableau de bord",
		ExtraData: map[string]any{
			"Username": username,
		},
	}

	RenderDashboardPage(w, data)
}

func HelpHandler(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := tools.CheckRequest(w, r, http.MethodGet); !ok {
		return
	}
	tools.RenderMergeTemplate(w, data.DashboardPageData{Routes: data.DefaultDashboardRoutes, PageTitle: "Aide"}, data.DefaultDashboarPath, data.DefaultDashboardName, data.DefaultDashboarPath, "help.html")
}
