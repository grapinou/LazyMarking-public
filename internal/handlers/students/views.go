package students

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableStudentPage(w http.ResponseWriter, dataPage data.StudentPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultStudentPathTemplate,
		data.DefaultStudentTemplateName.Table)
}

func RenderAddFormStudentPage(w http.ResponseWriter, dataPage data.StudentPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultStudentPathTemplate,
		data.DefaultStudentTemplateName.AddForm)
}

func RenderEditFormStudentPage(w http.ResponseWriter, dataPage data.StudentPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultStudentPathTemplate,
		data.DefaultStudentTemplateName.EditForm)
}

func RenderDeleteFormStudentPage(w http.ResponseWriter, dataPage data.StudentPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultStudentPathTemplate,
		data.DefaultStudentTemplateName.DeleteForm)
}

func RenderAddCSVFormStudentPage(w http.ResponseWriter, dataPage data.StudentPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultStudentPathTemplate,
		data.DefaultStudentTemplateName.AddCSVForm)
}
