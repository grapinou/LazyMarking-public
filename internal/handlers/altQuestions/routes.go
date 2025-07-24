package altquestions

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultQuestionRoutes
	altQuestionRoutes := data.DefaultAltQuestionRoutes

	mux.Handle("GET "+routes.AltQuestionsURL, login.CheckAuth(
		tools.HandlerWithDB(TableAltQuestionsHandler, queries)))

	mux.Handle("GET "+altQuestionRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormAltQuestionHandler, queries)))

	mux.Handle("POST "+altQuestionRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddAltQuestionHandler, queries)))

	mux.Handle("GET "+altQuestionRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormAltQuestionHandler, queries)))

	mux.Handle("POST "+altQuestionRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditAltQuestionHandler, queries)))
}
