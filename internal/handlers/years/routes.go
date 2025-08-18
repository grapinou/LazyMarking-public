package years

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	examRoutes := data.DefaultExamRoutes
	yearRoutes := data.DefaultYearRoutes

	mux.Handle("GET "+examRoutes.YearsURL, login.CheckAuth(
		tools.HandlerWithDB(TableYearsHandler, queries)))

	mux.Handle("GET "+yearRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormYearHandler, queries)))

	mux.Handle("POST "+yearRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddYearHandler, queries)))

	mux.Handle("GET "+yearRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormYearHandler, queries)))

	mux.Handle("POST "+yearRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditYearHandler, queries)))

	mux.Handle("GET "+yearRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormYearHandler, queries)))

	mux.Handle("POST "+yearRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteYearHandler, queries)))
}
