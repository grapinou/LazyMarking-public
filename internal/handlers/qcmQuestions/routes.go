package qcmquestions

import (
	"database/sql"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries, conn *sql.DB) {
	routes := data.DefaultQCMRoutes
	qcmQuestionsRoutes := data.DefaultQCMQuestionRoutes

	mux.Handle("GET "+routes.AddQuestionURL, login.CheckAuth(
		tools.HandlerWithDB(TableQCMQuestionsHandler, queries)))

	mux.Handle("GET "+qcmQuestionsRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormQCMQuestionHandler, queries)))

	mux.Handle("POST "+qcmQuestionsRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDBAndConn(AddQCMQuestionHandler, queries, conn)))

	mux.Handle("GET "+qcmQuestionsRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormQCMQuestionHandler, queries)))

	mux.Handle("POST "+qcmQuestionsRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDBAndConn(DeleteQCMQuestionHandler, queries, conn)))
}
