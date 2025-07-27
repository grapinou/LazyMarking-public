package themes

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableThemePage(w http.ResponseWriter, dataPage data.ThemePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultThemePathTemplate,
		data.DefaultThemeTemplateName.Table)
}

func RenderAddThemeFormPage(w http.ResponseWriter, dataPage data.ThemePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultThemePathTemplate,
		data.DefaultThemeTemplateName.AddForm)
}

func RenderEditFormThemePage(w http.ResponseWriter, dataPage data.ThemePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultThemePathTemplate,
		data.DefaultThemeTemplateName.EditForm)
}

func RenderDeleteFormThemePage(w http.ResponseWriter, dataPage data.ThemePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultThemePathTemplate,
		data.DefaultThemeTemplateName.DeleteForm)
}
