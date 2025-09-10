package marking

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	dashboardRoutes := data.DefaultDashboardRoutes
	markingRoutes := data.DefaultMarkingRoutes

	mux.Handle("GET "+dashboardRoutes.MarkingURL, login.CheckAuth(
		tools.HandlerWithDB(AddPdfFormMarkingHandler, queries)))

	mux.Handle("POST "+markingRoutes.ProcessingMarking, login.CheckAuth(
		tools.HandlerWithDB(ProcessingMarkingHandler, queries)))
}
