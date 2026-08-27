package logout

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if login.GetStore() == nil {
		http.Error(w, "Session store unavailable", http.StatusInternalServerError)
		return
	}
	session, err := login.GetStore().Get(r, "session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	// Invalide la session en supprimant les données et expirant le cookie
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		http.Error(w, "Failed to clear session", http.StatusInternalServerError)
		return
	}

	data := data.DefaultHomeRoutes
	// Redirection vers la page d’accueil ou de login
	http.Redirect(w, r, data.Home, http.StatusSeeOther)
}
