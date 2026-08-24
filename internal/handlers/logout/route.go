package logout

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("POST "+data.DefaultDashboardRoutes.LogoutURL, login.CheckAuth(http.HandlerFunc(LogoutHandler)))
}
