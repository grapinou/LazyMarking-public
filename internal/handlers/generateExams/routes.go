package generateexams

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	examRoutes := data.DefaultExamRoutes

	mux.Handle("GET "+examRoutes.GenerateExamPdf, login.CheckAuth(
		tools.HandlerWithDB(GenerateExamsHandler, queries)))
}
