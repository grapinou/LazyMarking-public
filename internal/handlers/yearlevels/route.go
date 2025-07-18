package yearlevels

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	yearLevelsRoutes := data.DefaultYearLevelRoutes

	mux.Handle(routes.YearLevelsURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				YearLevelsHandler(w, r, queries)
			}))))

	mux.Handle(yearLevelsRoutes.AddURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				AddYearLevelsFormHandler(w, r, queries)
			}))))

	mux.Handle("POST "+yearLevelsRoutes.AddURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				AddYearLevelsHandler(w, r, queries)
			}))))

	mux.Handle(yearLevelsRoutes.EditURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				EditYearLevelsFormHandler(w, r, queries)
			}))))

	mux.Handle("POST "+yearLevelsRoutes.EditURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				EditYearLevelsHandler(w, r, queries)
			}))))

	mux.Handle(yearLevelsRoutes.DeleteURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				DeleteFormYearLevelsHandler(w, r, queries)
			}))))

	mux.Handle("POST "+yearLevelsRoutes.DeleteURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				DeleteYearLevelsHandler(w, r, queries)
			}))))
}
