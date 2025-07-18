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
	AddURL:    "/dashboard/skills/add",
	EditURL:   "/dashboard/skills/edit",
	DeleteURL: "/dashboard/skills/delete",
}

type SkillPageData struct {
	Routes      DashboardRoutes
	SkillRoutes SkillRoutes
	PageTitle   string
	ExtraData   map[string]any
}
