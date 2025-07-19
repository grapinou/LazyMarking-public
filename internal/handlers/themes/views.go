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

func RenderAddThemeForm(w http.ResponseWriter, dataPage data.ThemePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultThemePathTemplate,
		data.DefaultThemeTemplateName.AddForm)
}

func RenderEditFormTheme(w http.ResponseWriter, dataPage data.ThemePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultThemePathTemplate,
		data.DefaultThemeTemplateName.EditForm)
}

func RenderDeleteFormTheme(w http.ResponseWriter, dataPage data.ThemePageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultThemePathTemplate,
		data.DefaultThemeTemplateName.DeleteForm)
}
