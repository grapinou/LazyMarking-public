package questions

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableQuestionPage(w http.ResponseWriter, dataPage data.QuestionPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultQuestionPathTemplate,
		data.DefaultQuestionTemplateName.Table)
}

func RenderAddFormQuestion(w http.ResponseWriter, dataPage data.QuestionPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultQuestionPathTemplate,
		data.DefaultQuestionTemplateName.AddForm)
}

func RenderEditFormQuestion(w http.ResponseWriter, dataPage data.QuestionPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultQuestionPathTemplate,
		data.DefaultQuestionTemplateName.EditForm)
}
