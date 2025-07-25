package altanswers

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableAltAnswerPage(w http.ResponseWriter, dataPage data.AltAnswerPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAltAnswerPathTemplate,
		data.DefaultAltAnswerTemplateName.Table)
}

func RenderAddFormAltAnswerPage(w http.ResponseWriter, dataPage data.AltAnswerPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAltAnswerPathTemplate,
		data.DefaultAltAnswerTemplateName.AddForm)
}

func RenderEditFormAltAnswerPage(w http.ResponseWriter, dataPage data.AltAnswerPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAltAnswerPathTemplate,
		data.DefaultAltAnswerTemplateName.EditForm)
}

func RenderDeleteFormAltAnswerPage(w http.ResponseWriter, dataPage data.AltAnswerPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAltAnswerPathTemplate,
		data.DefaultAltAnswerTemplateName.DeleteForm)
}
