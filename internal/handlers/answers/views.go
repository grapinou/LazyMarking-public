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

func RenderAddFormAnswer(w http.ResponseWriter, dataPage data.AnswerPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAnswerPathTemplate,
		data.DefaultAnswerTemplateName.AddForm)
}

func RenderEditFormAnswer(w http.ResponseWriter, dataPage data.AnswerPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAnswerPathTemplate,
		data.DefaultAnswerTemplateName.EditForm)
}

func RenderDeleteFormAnswer(w http.ResponseWriter, dataPage data.AnswerPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAnswerPathTemplate,
		data.DefaultAnswerTemplateName.DeleteForm)
}
