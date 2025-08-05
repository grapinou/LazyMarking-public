package altimages

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableAltImagePage(w http.ResponseWriter, dataPage data.AltImagePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAltImagePathTemplate,
		data.DefaultAltImageTemplateName.Table)
}

func RenderAddFormAltImagePage(w http.ResponseWriter, dataPage data.AltImagePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAltImagePathTemplate,
		data.DefaultAltImageTemplateName.AddForm)
}

func RenderEditFormAltImagePage(w http.ResponseWriter, dataPage data.AltImagePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAltImagePathTemplate,
		data.DefaultAltImageTemplateName.EditForm)
}

func RenderDeleteFormAltImagePage(w http.ResponseWriter, dataPage data.AltImagePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAltImagePathTemplate,
		data.DefaultAltImageTemplateName.DeleteForm)
}
