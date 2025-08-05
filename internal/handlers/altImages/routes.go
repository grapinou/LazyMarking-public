package altimages

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultAltQuestionRoutes
	altImageRoutes := data.DefaultAltImageRoutes

	mux.Handle("GET "+routes.AltImageURL, login.CheckAuth(
		tools.HandlerWithDB(TableAltImageHandler, queries)))

	mux.Handle("GET "+altImageRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormAltImageHandler, queries)))

	mux.Handle("POST "+altImageRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddAltImageHandler, queries)))

	mux.Handle("GET "+altImageRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormAltImageHandler, queries)))

	mux.Handle("POST "+altImageRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditAltImageHandler, queries)))

	mux.Handle("GET "+altImageRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormAltImageHandler, queries)))

	mux.Handle("POST "+altImageRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteAltImageHandler, queries)))
}
