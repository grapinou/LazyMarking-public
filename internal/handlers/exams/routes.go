package exams

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	examRoutes := data.DefaultExamRoutes

	mux.Handle("GET "+routes.ExamURL, login.CheckAuth(
		tools.HandlerWithDB(TableExamsHandler, queries)))

	mux.Handle("GET "+examRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormExamHandler, queries)))

	mux.Handle("POST "+examRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddExamHandler, queries)))

	mux.Handle("GET "+examRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormExamHandler, queries)))

	mux.Handle("POST "+examRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditExamHandler, queries)))

	mux.Handle("GET "+examRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormExamHandler, queries)))

	mux.Handle("POST "+examRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteExamHandler, queries)))
}
