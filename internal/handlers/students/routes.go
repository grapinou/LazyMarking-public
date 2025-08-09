package students

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	routes := data.DefaultDashboardRoutes
	// studentsRoutes := data.DefaultStudentRoutes

	mux.Handle("GET "+routes.StudentURL, login.CheckAuth(
		tools.HandlerWithDB(TableStudentsHandler, queries)))
}
