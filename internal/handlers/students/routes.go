package students

import (
	"database/sql"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries, conn *sql.DB) {
	routes := data.DefaultDashboardRoutes
	studentRoutes := data.DefaultStudentRoutes

	mux.Handle("GET "+routes.StudentURL, login.CheckAuth(
		tools.HandlerWithDB(TableStudentsHandler, queries)))

	mux.Handle("GET "+studentRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormStudentHandler, queries)))

	mux.Handle("POST "+studentRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDBAndConn(AddStudentHandler, queries, conn)))

	mux.Handle("GET "+studentRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormStudentHandler, queries)))

	mux.Handle("POST "+studentRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditStudentHandler, queries)))

	mux.Handle("GET "+studentRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormStudentHandler, queries)))

	mux.Handle("POST "+studentRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteStudentHandler, queries)))

	mux.Handle("GET "+studentRoutes.AddCSVURL, login.CheckAuth(
		tools.HandlerWithDB(AddCSVFormStudentHandler, queries)))

	mux.Handle("POST "+studentRoutes.AddCSVURL, login.CheckAuth(
		tools.HandlerWithDBAndConn(AddCSVStudentHandler, queries, conn)))

	mux.Handle("GET "+studentRoutes.DeleteAllStudentURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormAllStudentsHandler, queries)))

	mux.Handle("POST "+studentRoutes.DeleteAllStudentURL, login.CheckAuth(
		tools.HandlerWithDBAndConn(DeleteAllStudentsHandler, queries, conn)))
}
