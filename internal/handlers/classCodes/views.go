package classcodes

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableClassCodePage(w http.ResponseWriter, dataPage data.ClassCodePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultClassCodePathTemplate,
		data.DefaultClassCodeTemplateName.Table)
}

func RenderAddFormClassCodePage(w http.ResponseWriter, dataPage data.ClassCodePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultClassCodePathTemplate,
		data.DefaultClassCodeTemplateName.AddForm)
}

func RenderEditFormClassCodePage(w http.ResponseWriter, dataPage data.ClassCodePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultClassCodePathTemplate,
		data.DefaultClassCodeTemplateName.EditForm)
}

func RenderDeleteFormClassCodePage(w http.ResponseWriter, dataPage data.ClassCodePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultClassCodePathTemplate,
		data.DefaultClassCodeTemplateName.DeleteForm)
}
