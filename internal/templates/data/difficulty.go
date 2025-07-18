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
