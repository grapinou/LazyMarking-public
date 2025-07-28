package answers

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableAnswerPage(w http.ResponseWriter, dataPage data.AnswerPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAnswerPathTemplate,
		data.DefaultAnswerTemplateName.Table)
}

func RenderAddFormAnswerPage(w http.ResponseWriter, dataPage data.AnswerPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAnswerPathTemplate,
		data.DefaultAnswerTemplateName.AddForm)
}

func RenderEditFormAnswerPage(w http.ResponseWriter, dataPage data.AnswerPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAnswerPathTemplate,
		data.DefaultAnswerTemplateName.EditForm)
}

func RenderDeleteFormAnswerPage(w http.ResponseWriter, dataPage data.AnswerPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAnswerPathTemplate,
		data.DefaultAnswerTemplateName.DeleteForm)
}
