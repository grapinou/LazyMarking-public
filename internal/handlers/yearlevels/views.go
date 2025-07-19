package yearlevels

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableYearLevelPage(w http.ResponseWriter, dataPage data.YearLevelPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultYearLevelPathTemplate,
		data.DefaultYearLevelTemplateName.Table)
}

func RenderAddFormYearLevel(w http.ResponseWriter, dataPage data.YearLevelPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultYearLevelPathTemplate,
		data.DefaultYearLevelTemplateName.AddForm)
}

func RenderEditFormYearLevel(w http.ResponseWriter, dataPage data.YearLevelPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultYearLevelPathTemplate,
		data.DefaultYearLevelTemplateName.EditForm)
}

func RenderDeleteFormYearLevel(w http.ResponseWriter, dataPage data.YearLevelPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultYearLevelPathTemplate,
		data.DefaultYearLevelTemplateName.EditForm)
}
