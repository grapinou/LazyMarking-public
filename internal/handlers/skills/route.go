package skills

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	skillsRoutes := data.DefaultSkillRoutes

	mux.Handle(routes.SkillsURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				SkillsHandler(w, r, queries)
			}))))

	mux.Handle(skillsRoutes.AddURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				AddSkillsFormHandler(w, r, queries)
			}))))

	mux.Handle("POST "+skillsRoutes.AddURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				AddSkillsHandler(w, r, queries)
			}))))

	mux.Handle(skillsRoutes.EditURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				EditSkillsFormHandler(w, r, queries)
			}))))

	mux.Handle("POST "+skillsRoutes.EditURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				EditSkillsHandler(w, r, queries)
			}))))

	mux.Handle(skillsRoutes.DeleteURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				DeleteFormSkillsHandler(w, r, queries)
			}))))

	mux.Handle("POST "+skillsRoutes.DeleteURL, login.AuthMiddleware(
		login.ContextMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				DeleteSkillsHandler(w, r, queries)
			}))))
}
