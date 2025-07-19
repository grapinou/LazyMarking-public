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

type ThemeTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultThemeTemplateName = ThemeTemplateName{
	AddForm:    "add_form_theme.html",
	EditForm:   "edit_form_theme.html",
	DeleteForm: "delete_form_theme.html",
	Table:      "table_themes.html",
}

var DefaultThemePathTemplate = "internal/templates/themes/"
