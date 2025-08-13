package qcmquestions

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultQCMRoutes
	qcmQuestionsRoutes := data.DefaultQCMQuestionRoutes

	mux.Handle("GET "+routes.AddQuestionURL, login.CheckAuth(
		tools.HandlerWithDB(TableQCMQuestionsHandler, queries)))

	mux.Handle("GET "+qcmQuestionsRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormQCMQuestionHandler, queries)))
}
