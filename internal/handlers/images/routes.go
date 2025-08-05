package images

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultQuestionRoutes
	imageRoutes := data.DefaultImageRoutes

	mux.Handle("GET "+routes.ImageURL, login.CheckAuth(
		tools.HandlerWithDB(TableImageHandler, queries)))

	mux.Handle("GET "+imageRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormImageHandler, queries)))

	mux.Handle("POST "+imageRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddImageHandler, queries)))

	mux.Handle("GET "+imageRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormImageHandler, queries)))

	mux.Handle("POST "+imageRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditImageHandler, queries)))

	mux.Handle("GET "+imageRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormImageHandler, queries)))

	mux.Handle("POST "+imageRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteImageHandler, queries)))
}
