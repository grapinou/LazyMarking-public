package marking

import (
	"context"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries, appCtx context.Context) {
	dashboardRoutes := data.DefaultDashboardRoutes
	markingRoutes := data.DefaultMarkingRoutes

	mux.Handle("GET "+dashboardRoutes.MarkingURL, login.CheckAuth(
		tools.HandlerWithDB(AddPdfFormMarkingHandler, queries)))

	mux.Handle("POST "+markingRoutes.ProcessingMarking, login.CheckAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ProcessingMarkingHandler(w, r, queries, appCtx)
	})))

	mux.Handle("GET "+markingRoutes.ProgressMarking, login.CheckAuth(
		tools.HandlerWithDB(ProgressMarkingHandler, queries)))

	mux.Handle("GET "+markingRoutes.SuccessURL, login.CheckAuth(
		tools.HandlerWithDB(SuccessMarkingProcessingHandler, queries)))

	mux.Handle("GET "+markingRoutes.ServePDF, login.CheckAuth(
		tools.HandlerWithDB(ServeFullMarkingPdfHandler, queries)))
}
