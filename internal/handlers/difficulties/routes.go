package difficulties

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	questionsroutes := data.DefaultQuestionRoutes
	difficultiesRoutes := data.DefaultDifficultyRoutes

	mux.Handle("GET "+questionsroutes.DifficultiesURL, login.CheckAuth(
		tools.HandlerWithDB(TableDifficultiesHandler, queries)))

	mux.Handle("GET "+difficultiesRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormDifficultyHandler, queries)))

	mux.Handle("POST "+difficultiesRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddDifficultyHandler, queries)))

	mux.Handle("GET "+difficultiesRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormDifficultyHandler, queries)))

	mux.Handle("POST "+difficultiesRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditDifficultyHandler, queries)))

	mux.Handle("GET "+difficultiesRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormDifficultyHandler, queries)))

	mux.Handle("POST "+difficultiesRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteDifficultyHandler, queries)))
}
