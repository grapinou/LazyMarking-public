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

func RenderAddFormSubject(w http.ResponseWriter, dataPage data.SubjectPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultSubjectPathTemplate,
		data.DefaultSubjectTemplateName.AddForm)
}

func RenderEditFormSubject(w http.ResponseWriter, dataPage data.SubjectPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultSubjectPathTemplate,
		data.DefaultSubjectTemplateName.EditForm)
}

func RenderDeleteFormSubject(w http.ResponseWriter, dataPage data.SubjectPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultSubjectPathTemplate,
		data.DefaultSubjectTemplateName.DeleteForm)
}
