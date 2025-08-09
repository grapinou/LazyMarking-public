package altpreview

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	altQuestionsroutes := data.DefaultAltQuestionRoutes
	altPreviewRoute := data.DefaultAltPreviewAltQuestionRoutes

	mux.Handle("GET "+altQuestionsroutes.AltPreviewURL, login.CheckAuth(
		tools.HandlerWithDB(AltPreviewAltQuestionHandler, queries)))

	mux.Handle("GET "+altPreviewRoute.AltPreviewAltQuestion, login.CheckAuth(
		tools.HandlerWithDB(AltServePreviewPDFHandler, queries)))
}
