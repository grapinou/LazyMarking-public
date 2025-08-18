package years

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableYearPage(w http.ResponseWriter, dataPage data.YearPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultYearPathTemplate,
		data.DefaultYearTemplateName.Table)
}

func RenderAddFormYearPage(w http.ResponseWriter, dataPage data.YearPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultYearPathTemplate,
		data.DefaultYearTemplateName.AddForm)
}

func RenderEditFormYearPage(w http.ResponseWriter, dataPage data.YearPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultYearPathTemplate,
		data.DefaultYearTemplateName.EditForm)
}

func RenderDeleteFormYearPage(w http.ResponseWriter, dataPage data.YearPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultYearPathTemplate,
		data.DefaultYearTemplateName.DeleteForm)
}
