package skills

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	questionsroutes := data.DefaultQuestionRoutes
	skillsRoutes := data.DefaultSkillRoutes

	mux.Handle("GET "+questionsroutes.SkillsURL, login.CheckAuth(
		tools.HandlerWithDB(TableSkillsHandler, queries)))

	mux.Handle("GET "+skillsRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormSkillHandler, queries)))

	mux.Handle("POST "+skillsRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddSkillHandler, queries)))

	mux.Handle("GET "+skillsRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormSkillHandler, queries)))

	mux.Handle("POST "+skillsRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditSkillHandler, queries)))

	mux.Handle("GET "+skillsRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormSkillHandler, queries)))

	mux.Handle("POST "+skillsRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteSkillHandler, queries)))
}
