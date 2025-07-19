package yearlevels

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	yearLevelsRoutes := data.DefaultYearLevelRoutes

	mux.Handle("GET "+routes.YearLevelsURL, login.CheckAuth(
		tools.HandlerWithDB(TableYearLevelsHandler, queries)))

	mux.Handle("GET "+yearLevelsRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormYearLevelHandler, queries)))

	mux.Handle("POST "+yearLevelsRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddYearLevelHandler, queries)))

	mux.Handle("GET "+yearLevelsRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormYearLevelHandler, queries)))

	mux.Handle("POST "+yearLevelsRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditYearLevelHandler, queries)))

	mux.Handle("GET "+yearLevelsRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormYearLevelHandler, queries)))

	mux.Handle("POST "+yearLevelsRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteYearLevelHandler, queries)))
}
