package themes

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	themesRoutes := data.DefaultThemeRoutes

	mux.Handle(routes.ThemesURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ThemesHandler(w, r, queries)
			}))))

	mux.Handle(themesRoutes.AddURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				AddThemesFormHandler(w, r, queries)
			}))))

	mux.Handle("POST "+themesRoutes.AddURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				AddThemesHandler(w, r, queries)
			}))))

	mux.Handle(themesRoutes.EditURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				EditThemesFormHandler(w, r, queries)
			}))))

	mux.Handle("POST "+themesRoutes.EditURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				EditThemesHandler(w, r, queries)
			}))))

	mux.Handle(themesRoutes.DeleteURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				DeleteFormThemesHandler(w, r, queries)
			}))))

	mux.Handle("POST "+themesRoutes.DeleteURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				DeleteThemesHandler(w, r, queries)
			}))))
}
