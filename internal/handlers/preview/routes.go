package preview

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	questionsroutes := data.DefaultQuestionRoutes
	previewRoute := data.DefaultPreviewQuestionRoutes

	mux.Handle("GET "+questionsroutes.PreviewURL, login.CheckAuth(
		tools.HandlerWithDB(PreviewQuestionHandler, queries)))

	mux.Handle("GET "+previewRoute.PreviewQuestion, login.CheckAuth(
		tools.HandlerWithDB(ServePreviewPDFHandler, queries)))
}
