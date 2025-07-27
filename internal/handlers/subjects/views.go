package subjects

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableSubjectPage(w http.ResponseWriter, dataPage data.SubjectPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultSubjectPathTemplate,
		data.DefaultSubjectTemplateName.Table)
}

func RenderAddFormSubjectPage(w http.ResponseWriter, dataPage data.SubjectPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultSubjectPathTemplate,
		data.DefaultSubjectTemplateName.AddForm)
}

func RenderEditFormSubjectPage(w http.ResponseWriter, dataPage data.SubjectPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultSubjectPathTemplate,
		data.DefaultSubjectTemplateName.EditForm)
}

func RenderDeleteFormSubjectPage(w http.ResponseWriter, dataPage data.SubjectPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultSubjectPathTemplate,
		data.DefaultSubjectTemplateName.DeleteForm)
}
