package data

type DifficultyRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type DifficultyActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultDifficultyRoutes = DifficultyRoutes{
	AddURL:    "/dashboard/difficulties/add",
	EditURL:   "/dashboard/difficulties/edit",
	DeleteURL: "/dashboard/difficulties/delete",
}

type DifficultyPageData struct {
	Routes           DashboardRoutes
	DifficultyRoutes DifficultyRoutes
	PageTitle        string
	ExtraData        map[string]any
}

type DifficultyTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultDifficultyTemplateName = DifficultyTemplateName{
	AddForm:    "add_form_difficulty.html",
	EditForm:   "edit_form_difficulty.html",
	DeleteForm: "delete_form_difficulty.html",
	Table:      "table_difficulties.html",
}

var DefaultDifficultyPathTemplate = "internal/templates/difficulties/"
