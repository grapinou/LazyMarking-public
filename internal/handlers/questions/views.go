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

func RenderAddFormQuestionPage(w http.ResponseWriter, dataPage data.QuestionPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultQuestionPathTemplate,
		data.DefaultQuestionTemplateName.AddForm)
}

func RenderEditFormQuestionPage(w http.ResponseWriter, dataPage data.QuestionPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultQuestionPathTemplate,
		data.DefaultQuestionTemplateName.EditForm)
}

func RenderDeleteFormQuestionPage(w http.ResponseWriter, dataPage data.QuestionPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultQuestionPathTemplate,
		data.DefaultQuestionTemplateName.DeleteForm)
}
