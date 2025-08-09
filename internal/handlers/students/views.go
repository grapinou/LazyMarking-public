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
