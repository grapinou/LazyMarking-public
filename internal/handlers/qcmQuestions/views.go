package qcmquestions

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableQCMQuestionPage(w http.ResponseWriter, dataPage data.QCMQuestionPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultQCMQuestionPathTemplate,
		data.DefaultQCMQuestionTemplateName.Table)
}

func RenderAddFormQCMQuestionPage(w http.ResponseWriter, dataPage data.QCMQuestionPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultQCMQuestionPathTemplate,
		data.DefaultQCMQuestionTemplateName.AddForm)
}

func RenderDeleteFormQCMQuestionPage(w http.ResponseWriter, dataPage data.QCMQuestionPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultQCMQuestionPathTemplate,
		data.DefaultQCMQuestionTemplateName.DeleteForm)
}
