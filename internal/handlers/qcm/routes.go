package qcm

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	dashboardRoutes := data.DefaultDashboardRoutes
	QCMRoutes := data.DefaultQCMRoutes

	mux.Handle("GET "+dashboardRoutes.QcmURL, login.CheckAuth(
		tools.HandlerWithDB(TableQCMHandler, queries)))

	mux.Handle("GET "+QCMRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddFormQCMHandler, queries)))

	mux.Handle("POST "+QCMRoutes.AddURL, login.CheckAuth(
		tools.HandlerWithDB(AddQCMHandler, queries)))

	mux.Handle("GET "+QCMRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditFormQCMHandler, queries)))

	mux.Handle("POST "+QCMRoutes.EditURL, login.CheckAuth(
		tools.HandlerWithDB(EditQCMHandler, queries)))

	mux.Handle("GET "+QCMRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteFormQCMHandler, queries)))

	mux.Handle("POST "+QCMRoutes.DeleteURL, login.CheckAuth(
		tools.HandlerWithDB(DeleteQCMHandler, queries)))
}
