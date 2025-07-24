package altquestions

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableAltQuestionPage(w http.ResponseWriter, dataPage data.AltQuestionPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAltQuestionPathTemplate,
		data.DefaultAltQuestionTemplateName.Table)
}

func RenderAddFormAltQuestionPage(w http.ResponseWriter, dataPage data.AltQuestionPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAltQuestionPathTemplate,
		data.DefaultAltQuestionTemplateName.AddForm)
}

func RenderEditFormAltQuestionPage(w http.ResponseWriter, dataPage data.AltQuestionPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultAltQuestionPathTemplate,
		data.DefaultAltQuestionTemplateName.EditForm)
}
