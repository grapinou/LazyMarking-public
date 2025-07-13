package dashboard

import (
	"log"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/login"
)

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := login.GetUserID(r)
	if !ok {
		http.Error(w, "Authentification failed. Unauthorized.", http.StatusUnauthorized)
		return
	}

	log.Println("Logged as : ", userID)
}
