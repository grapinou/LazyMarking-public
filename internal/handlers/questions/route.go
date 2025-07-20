package questions

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	questionsRoutes := data.DefaultQuestionRoutes

	mux.Handle("GET "+routes.QuestionsURL, login.CheckAuth(
		tools.HandlerWithDB(TableQuestionsHandler, queries)))

	mux.Handle("GET "+questionsRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormQuestionsHandler, queries)))

	mux.Handle("POST "+questionsRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddQuestionsHandler, queries)))
}
