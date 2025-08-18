package periods

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	examRoutes := data.DefaultExamRoutes
	periodRoutes := data.DefaultPeriodRoutes

	mux.Handle("GET "+examRoutes.PeriodsURL, login.CheckAuth(
		tools.HandlerWithDB(TablePeriodsHandler, queries)))

	mux.Handle("GET "+periodRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormPeriodHandler, queries)))

	mux.Handle("POST "+periodRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddPeriodHandler, queries)))

	mux.Handle("GET "+periodRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormPeriodHandler, queries)))

	mux.Handle("POST "+periodRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditPeriodHandler, queries)))

	mux.Handle("GET "+periodRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormPeriodHandler, queries)))

	mux.Handle("POST "+periodRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeletePeriodHandler, queries)))
}
