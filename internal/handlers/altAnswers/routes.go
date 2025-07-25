package altanswers

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	altQuestionRoutes := data.DefaultAltQuestionRoutes
	altAnswerRoutes := data.DefaultAltAnswerRoutes

	mux.Handle("GET "+altQuestionRoutes.AltAnswersURL, login.CheckAuth(
		tools.HandlerWithDB(TableAltAnswersHandler, queries)))

	mux.Handle("GET "+altAnswerRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormAltAnswerHandler, queries)))

	mux.Handle("POST "+altAnswerRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddAltAnswerHandler, queries)))

	mux.Handle("GET "+altAnswerRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormAltAnswerHandler, queries)))

	mux.Handle("POST "+altAnswerRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditAltAnswerHandler, queries)))

	mux.Handle("GET "+altAnswerRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormAltAnswerHandler, queries)))

	mux.Handle("POST "+altAnswerRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteAltAnswerHandler, queries)))
}
