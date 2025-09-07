package generateexams

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderProcessingStudentsPage(w http.ResponseWriter, dataPage data.GenerateExamPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultGenerateExamPathTemplate,
		data.DefaultGenerateExamTemplateName.ProcessingStudents)
}

func RenderSuccessProcessing(w http.ResponseWriter, dataPage data.GenerateExamPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultGenerateExamPathTemplate,
		data.DefaultGenerateExamTemplateName.SuccessProcessing)
}
