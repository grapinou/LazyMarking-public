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
	generateRoutes := data.DefaultGenerateExamRoutes

	mux.Handle("GET "+examRoutes.GenerateExamPdf, login.CheckAuth(
		tools.HandlerWithDB(GenerateExamsHandler, queries)))

	mux.Handle("GET "+generateRoutes.ProcessingStudents, login.CheckAuth(
		tools.HandlerWithDB(GetExamProgressPageHandler, queries)))

	mux.Handle("GET "+generateRoutes.PdfExam, login.CheckAuth(
		tools.HandlerWithDB(ServeFullPdfExamHandler, queries)))

	mux.Handle("GET "+examRoutes.GenerateMiniPdf, login.CheckAuth(
		tools.HandlerWithDB(GenerateMiniPDFHandler, queries)))

	mux.Handle("GET "+generateRoutes.MiniQCMLandscape, login.CheckAuth(
		tools.HandlerWithDB(ServeMiniPDFHandler, queries)))
}
