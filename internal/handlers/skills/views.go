package skills

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderTableSkillPage(w http.ResponseWriter, dataPage data.SkillPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultSkillPathTemplate,
		data.DefaultSkillTemplateName.Table)
}

func RenderAddFormSkill(w http.ResponseWriter, dataPage data.SkillPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultSkillPathTemplate,
		data.DefaultSkillTemplateName.AddForm)
}

func RenderEditFormSkill(w http.ResponseWriter, dataPage data.SkillPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultSkillPathTemplate,
		data.DefaultSkillTemplateName.EditForm)
}

func RenderDeleteFormSkill(w http.ResponseWriter, dataPage data.SkillPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultSkillPathTemplate,
		data.DefaultSkillTemplateName.DeleteForm)
}
