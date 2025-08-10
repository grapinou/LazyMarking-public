package studentclasscode

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableStudentClassCodesPage(w http.ResponseWriter, dataPage data.StudentClassCodePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultStudentClassCodePathTemplate,
		data.DefaultStudentClassCodeTemplateName.Table)
}

func RenderAddFormStudentClassCodePage(w http.ResponseWriter, dataPage data.StudentClassCodePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultStudentClassCodePathTemplate,
		data.DefaultStudentClassCodeTemplateName.AddForm)
}
