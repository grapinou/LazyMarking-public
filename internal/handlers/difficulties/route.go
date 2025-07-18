package difficulties

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	difficultiesRoutes := data.DefaultDifficultyRoutes

	mux.Handle(routes.DifficultiesURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				DifficultiesHandler(w, r, queries)
			}))))

	mux.Handle(difficultiesRoutes.AddURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				AddDifficultiesFormHandler(w, r, queries)
			}))))

	mux.Handle("POST "+difficultiesRoutes.AddURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				AddDifficultiesHandler(w, r, queries)
			}))))

	mux.Handle(difficultiesRoutes.EditURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				EditDifficultiesFormHandler(w, r, queries)
			}))))

	mux.Handle("POST "+difficultiesRoutes.EditURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				EditDifficultiesHandler(w, r, queries)
			}))))

	mux.Handle(difficultiesRoutes.DeleteURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				DeleteFormDifficultiesHandler(w, r, queries)
			}))))

	mux.Handle("POST "+difficultiesRoutes.DeleteURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				DeleteDifficultiesHandler(w, r, queries)
			}))))
}
