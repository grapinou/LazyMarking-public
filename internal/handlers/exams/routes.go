package exams

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	//examRoutes := data.DefaultExamRoutes

	mux.Handle("GET "+routes.ExamURL, login.CheckAuth(
		tools.HandlerWithDB(TableExamsHandler, queries)))
}
