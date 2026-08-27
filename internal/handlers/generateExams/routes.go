package generateexams

import (
	"context"
	"net/http"
	"sync"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries, appCtx context.Context, backgroundJobs *sync.WaitGroup) {
	examRoutes := data.DefaultExamRoutes
	generateRoutes := data.DefaultGenerateExamRoutes

	mux.Handle("POST "+examRoutes.GenerateExamPdf, login.CheckAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		GenerateExamsHandler(w, r, queries, appCtx, backgroundJobs)
	})))

	mux.Handle("GET "+generateRoutes.ProcessingStudents, login.CheckAuth(
		tools.HandlerWithDB(GetExamProgressPageHandler, queries)))

	mux.Handle("GET "+generateRoutes.PdfExam, login.CheckAuth(
		tools.HandlerWithDB(ServeFullPdfExamHandler, queries)))

	mux.Handle("POST "+examRoutes.GenerateMiniPdf, login.CheckAuth(
		tools.HandlerWithDB(GenerateMiniPDFHandler, queries)))

	mux.Handle("GET "+generateRoutes.MiniQCMLandscape, login.CheckAuth(
		tools.HandlerWithDB(ServeMiniPDFHandler, queries)))
}
