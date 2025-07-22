package answers

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	questionsroutes := data.DefaultQuestionRoutes
	answerRoutes := data.DefaultAnswerRoutes

	mux.Handle("GET "+questionsroutes.AnswersURL, login.CheckAuth(
		tools.HandlerWithDB(TableAnswersHandler, queries)))

	mux.Handle("GET "+answerRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormAnswerHandler, queries)))

	mux.Handle("POST "+answerRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddAnswerHandler, queries)))

	mux.Handle("GET "+answerRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormAnswerHandler, queries)))

	mux.Handle("POST "+answerRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditAnswerHandler, queries)))

	mux.Handle("GET "+answerRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormAnswerHandler, queries)))

	mux.Handle("POST "+answerRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteAnswerHandler, queries)))
}
