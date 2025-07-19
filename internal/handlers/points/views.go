package points

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTablePointPage(w http.ResponseWriter, dataPage data.PointPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultPointPathTemplate,
		data.DefaultPointTemplateName.Table)
}

func RenderAddFormPoint(w http.ResponseWriter, dataPage data.PointPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultPointPathTemplate,
		data.DefaultPointTemplateName.AddForm)
}

func RenderEditFormPoint(w http.ResponseWriter, dataPage data.PointPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultPointPathTemplate,
		data.DefaultPointTemplateName.EditForm)
}

func RenderDeleteFormPoint(w http.ResponseWriter, dataPage data.PointPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultPointPathTemplate,
		data.DefaultPointTemplateName.DeleteForm)
}
