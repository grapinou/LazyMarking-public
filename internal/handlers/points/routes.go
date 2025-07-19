package points

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	pointsRoutes := data.DefaultPointRoutes

	mux.Handle("GET "+routes.PointsURL, login.CheckAuth(
		tools.HandlerWithDB(TablePointsHandler, queries)))

	mux.Handle("GET "+pointsRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormPointHandler, queries)))

	mux.Handle("POST "+pointsRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddPointHandler, queries)))

	mux.Handle("GET "+pointsRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormPointHandler, queries)))

	mux.Handle("POST "+pointsRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditPointHandler, queries)))

	mux.Handle("GET "+pointsRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormPointHandler, queries)))

	mux.Handle("POST "+pointsRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeletePointHandler, queries)))
}
