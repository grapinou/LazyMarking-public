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
	// previewQCMRoute := data.DefaultPreviewQCMRoutes

	mux.Handle("GET "+qcmRoutes.PreviewURL, login.CheckAuth(
		tools.HandlerWithDB(PreviewQCMHandler, queries)))

	/*
		mux.Handle("GET "+previewRoute.PreviewQuestion, login.CheckAuth(
			tools.HandlerWithDB(ServePreviewPDFHandler, queries)))
	*/
}
