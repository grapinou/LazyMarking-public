package qcm

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableQCMPage(w http.ResponseWriter, dataPage data.QCMPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultQCMPathTemplate,
		data.DefaultQCMTemplateName.Table)
}

func RenderAddFormQCMPage(w http.ResponseWriter, dataPage data.QCMPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultQCMPathTemplate,
		data.DefaultQCMTemplateName.AddForm)
}

func RenderEditFormQCMPage(w http.ResponseWriter, dataPage data.QCMPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultQCMPathTemplate,
		data.DefaultQCMTemplateName.EditForm)
}

func RenderDeleteFormQCMPage(w http.ResponseWriter, dataPage data.QCMPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultQCMPathTemplate,
		data.DefaultQCMTemplateName.DeleteForm)
}
