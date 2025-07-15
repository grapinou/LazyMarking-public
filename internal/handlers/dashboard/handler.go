package dashboard

import (
	"log"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, username, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	log.Println("Logged as : ", userID)
	log.Println("username : ", username)

	data := data.DashboardPageData{
		Routes:    data.DefaultDashboardRoutes,
		PageTitle: "Dashboard",
		ExtraData: map[string]any{
			"Username": username,
		},
	}

	RenderDashboardPage(w, data)
}
