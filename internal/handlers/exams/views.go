package exams

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableExamPage(w http.ResponseWriter, dataPage data.ExamPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultExamPathTemplate,
		data.DefaultExamTemplateName.Table)
}

func RenderAddFormExamPage(w http.ResponseWriter, dataPage data.ExamPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultExamPathTemplate,
		data.DefaultExamTemplateName.AddForm)
}

func RenderEditFormExamPage(w http.ResponseWriter, dataPage data.ExamPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultExamPathTemplate,
		data.DefaultExamTemplateName.EditForm)
}

func RenderDeleteFormExamPage(w http.ResponseWriter, dataPage data.ExamPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultExamPathTemplate,
		data.DefaultExamTemplateName.DeleteForm)
}
