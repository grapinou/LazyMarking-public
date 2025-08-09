package classcodes

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	studentsroutes := data.DefaultStudentRoutes
	classCodesRoutes := data.DefaultClassCodeRoutes

	mux.Handle("GET "+studentsroutes.ClassCodesURL, login.CheckAuth(
		tools.HandlerWithDB(TableClassCodesHandler, queries)))

	mux.Handle("GET "+classCodesRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormClassCodeHandler, queries)))

	mux.Handle("POST "+classCodesRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddClassCodeHandler, queries)))

	mux.Handle("GET "+classCodesRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormClassCodeHandler, queries)))

	mux.Handle("POST "+classCodesRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditClassCodeHandler, queries)))

	mux.Handle("GET "+classCodesRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormClassCodeHandler, queries)))

	mux.Handle("POST "+classCodesRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteClassCodeHandler, queries)))
}
