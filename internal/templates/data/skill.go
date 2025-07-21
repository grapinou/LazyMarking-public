package data

type SkillRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type SkillActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultSkillRoutes = SkillRoutes{
	AddURL:    "/dashboard/questions/skills/add",
	EditURL:   "/dashboard/questions/skills/edit",
	DeleteURL: "/dashboard/questions/skills/delete",
}

type SkillPageData struct {
	Routes      DashboardRoutes
	SkillRoutes SkillRoutes
	PageTitle   string
	ExtraData   map[string]any
}

type SkillTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultSkillTemplateName = SkillTemplateName{
	AddForm:    "add_form_skill.html",
	EditForm:   "edit_form_skill.html",
	DeleteForm: "delete_form_skill.html",
	Table:      "table_skills.html",
}

var DefaultSkillPathTemplate = "internal/templates/skills/"
