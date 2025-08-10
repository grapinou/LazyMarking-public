package studentclasscode

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	studentRoutes := data.DefaultStudentRoutes
	studentClassCodeRoutes := data.DefaultStudentClassCodeRoutes

	mux.Handle("GET "+studentRoutes.StudentClassCodesURL, login.CheckAuth(
		tools.HandlerWithDB(TableStudentClassCodesHandler, queries)))

	mux.Handle("GET "+studentClassCodeRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormStudentClassCodeHandler, queries)))

	mux.Handle("POST "+studentClassCodeRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddStudentClassCodeHandler, queries)))

	mux.Handle("GET "+studentClassCodeRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteStudentClassCodeHandler, queries)))
}
