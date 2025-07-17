package data

type ThemeRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type ThemeActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultThemeRoutes = ThemeRoutes{
	AddURL:    "/dashboard/themes/add",
	EditURL:   "/dashboard/themes/edit",
	DeleteURL: "/dashboard/themes/delete",
}

type ThemePageData struct {
	Routes      DashboardRoutes
	ThemeRoutes ThemeRoutes
	PageTitle   string
	ExtraData   map[string]any
}
