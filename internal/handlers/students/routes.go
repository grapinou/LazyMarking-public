package students

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	studentRoutes := data.DefaultStudentRoutes

	mux.Handle("GET "+routes.StudentURL, login.CheckAuth(
		tools.HandlerWithDB(TableStudentsHandler, queries)))

	mux.Handle("GET "+studentRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormStudentHandler, queries)))

	mux.Handle("POST "+studentRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddStudentHandler, queries)))

	mux.Handle("GET "+studentRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormStudentHandler, queries)))

	mux.Handle("POST "+studentRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditStudentHandler, queries)))

	mux.Handle("GET "+studentRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormStudentHandler, queries)))

	mux.Handle("POST "+studentRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteStudentHandler, queries)))

	mux.Handle("GET "+studentRoutes.StudentClassCodesURL, login.CheckAuth(
		tools.HandlerWithDB(AddCSVFormStudentHandler, queries)))

	mux.Handle("POST "+studentRoutes.StudentClassCodesURL, login.CheckAuth(
		tools.HandlerWithDB(AddCSVStudentHandler, queries)))
}
