package themes

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	themesRoutes := data.DefaultThemeRoutes

	mux.Handle("GET "+routes.ThemesURL, login.CheckAuth(
		tools.HandlerWithDB(TableThemesHandler, queries)))

	mux.Handle("GET "+themesRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormThemeHandler, queries)))

	mux.Handle("POST "+themesRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddThemeHandler, queries)))

	mux.Handle("GET "+themesRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormThemeHandler, queries)))

	mux.Handle("POST "+themesRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditThemeHandler, queries)))

	mux.Handle("GET "+themesRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormThemeHandler, queries)))

	mux.Handle("POST "+themesRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteThemeHandler, queries)))
}
