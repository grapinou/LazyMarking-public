package images

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableImagePage(w http.ResponseWriter, dataPage data.ImagePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultImagePathTemplate,
		data.DefaultImageTemplateName.Table)
}

func RenderAddFormImagePage(w http.ResponseWriter, dataPage data.ImagePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultImagePathTemplate,
		data.DefaultImageTemplateName.AddForm)
}

func RenderEditFormImagePage(w http.ResponseWriter, dataPage data.ImagePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultImagePathTemplate,
		data.DefaultImageTemplateName.EditForm)
}

func RenderDeleteFormImagePage(w http.ResponseWriter, dataPage data.ImagePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultImagePathTemplate,
		data.DefaultImageTemplateName.DeleteForm)
}
