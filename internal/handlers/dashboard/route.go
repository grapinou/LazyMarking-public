package dashboard

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/dashboard/help", login.AuthMiddleware(login.ContextMiddleware(http.HandlerFunc(HelpHandler))))
	mux.Handle(data.DefaultDashboardRoutes.DashboardURL, login.AuthMiddleware(login.ContextMiddleware(http.HandlerFunc(DashboardHandler))))
}
