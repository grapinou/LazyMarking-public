package difficulties

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableDifficultyPage(w http.ResponseWriter, dataPage data.DifficultyPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultDifficultyPathTemplate,
		data.DefaultDifficultyTemplateName.Table)
}

func RenderAddFormDifficulty(w http.ResponseWriter, dataPage data.DifficultyPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultDifficultyPathTemplate,
		data.DefaultDifficultyTemplateName.AddForm)
}

func RenderEditFormDifficulty(w http.ResponseWriter, dataPage data.DifficultyPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultDifficultyPathTemplate,
		data.DefaultDifficultyTemplateName.EditForm)
}

func RenderDeleteFormDifficulty(w http.ResponseWriter, dataPage data.DifficultyPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultDifficultyPathTemplate,
		data.DefaultDifficultyTemplateName.DeleteForm)
}
