package subjects

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	subjectsRoutes := data.DefaultSubjectRoutes

	mux.Handle(routes.SubjectsURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				SubjectsHandler(w, r, queries)
			}))))

	mux.Handle(subjectsRoutes.AddURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				AddSubjectsFormHandler(w, r, queries)
			}))))

	mux.Handle("POST "+subjectsRoutes.AddURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				AddSubjectsHandler(w, r, queries)
			}))))

	mux.Handle(subjectsRoutes.EditURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				EditSubjectsFormHandler(w, r, queries)
			}))))

	mux.Handle("POST "+subjectsRoutes.EditURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				EditSubjectsHandler(w, r, queries)
			}))))

	mux.Handle(subjectsRoutes.DeleteURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				DeleteFormSubjectsHandler(w, r, queries)
			}))))

	mux.Handle("POST "+subjectsRoutes.DeleteURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				DeleteSubjectsHandler(w, r, queries)
			}))))
}
