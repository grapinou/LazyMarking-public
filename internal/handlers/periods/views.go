package periods

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTablePeriodPage(w http.ResponseWriter, dataPage data.PeriodPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultPeriodPathTemplate,
		data.DefaultPeriodTemplateName.Table)
}

func RenderAddFormPeriodPage(w http.ResponseWriter, dataPage data.PeriodPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultPeriodPathTemplate,
		data.DefaultPeriodTemplateName.AddForm)
}

func RenderEditFormPeriodPage(w http.ResponseWriter, dataPage data.PeriodPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultPeriodPathTemplate,
		data.DefaultPeriodTemplateName.EditForm)
}

func RenderDeleteFormPeriodPage(w http.ResponseWriter, dataPage data.PeriodPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultPeriodPathTemplate,
		data.DefaultPeriodTemplateName.DeleteForm)
}
