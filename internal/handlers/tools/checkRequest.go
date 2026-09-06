package tools

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/login"
)

// CheckRequest valide la méthode HTTP et extrait les infos utilisateur depuis le contexte.
//
// Si la méthode est incorrecte ou si l'utilisateur n'est pas authentifié, une erreur HTTP est automatiquement renvoyée.
// Retourne userID, userName, ok
func CheckRequest(w http.ResponseWriter, r *http.Request, expectedMethod string) (int64, string, bool) {
	if r.Method != expectedMethod {
		http.Error(w, "Cette méthode de requête n’est pas autorisée.", http.StatusMethodNotAllowed)
		return 0, "", false
	}

	userID, userName, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Authentification requise.", http.StatusUnauthorized)
		return 0, "", false
	}

	return userID, userName, true
}
