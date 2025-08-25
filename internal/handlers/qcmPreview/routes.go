package qcmpreview

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux, queries *db.Queries) {
	qcmRoutes := data.DefaultQCMRoutes
	previewQCMRoute := data.DefaultPreviewQCMRoutes

	mux.Handle("GET "+qcmRoutes.PreviewURL, login.CheckAuth(
		tools.HandlerWithDB(PreviewQCMHandler, queries)))

	mux.Handle("GET "+previewQCMRoute.PreviewQCM, login.CheckAuth(
		tools.HandlerWithDB(ServePreviewQCMPDFHandler, queries)))

	mux.Handle("GET "+qcmRoutes.PreviewLandscapeURL, login.CheckAuth(
		tools.HandlerWithDB(PreviewQCMLandscapeHandler, queries)))

	mux.Handle("GET "+previewQCMRoute.PreviewLandscapeQCM, login.CheckAuth(
		tools.HandlerWithDB(ServePreviewQCMLandscapePDFHandler, queries)))
}
