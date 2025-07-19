package subjects

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	subjectsRoutes := data.DefaultSubjectRoutes

	mux.Handle("GET "+routes.SubjectsURL, login.CheckAuth(
		tools.HandlerWithDB(TableSubjectsHandler, queries)))

	mux.Handle("GET "+subjectsRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormSubjectHandler, queries)))

	mux.Handle("POST "+subjectsRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddSubjectHandler, queries)))

	mux.Handle("GET "+subjectsRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormSubjectHandler, queries)))

	mux.Handle("POST "+subjectsRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditSubjectHandler, queries)))

	mux.Handle("GET "+subjectsRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormSubjectHandler, queries)))

	mux.Handle("POST "+subjectsRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteSubjectHandler, queries)))
}
