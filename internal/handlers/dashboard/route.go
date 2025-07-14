package dashboard

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/login"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/dashboard", login.AuthMiddleware(http.HandlerFunc(DashboardHandler)))
}
