package data

type YearLevelRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type YearYevelActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultYearLevelRoutes = YearLevelRoutes{
	AddURL:    "/dashboard/yearlevels/add",
	EditURL:   "/dashboard/yearlevels/edit",
	DeleteURL: "/dashboard/yearlevels/delete",
}

type YearLevelPageData struct {
	Routes          DashboardRoutes
	YearLevelRoutes YearLevelRoutes
	PageTitle       string
	ExtraData       map[string]any
}

type YearLevelTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultYearLevelTemplateName = YearLevelTemplateName{
	AddForm:    "add_form_yearlevel.html",
	EditForm:   "edit_form_yearlevel.html",
	DeleteForm: "delete_form_yearlevel.html",
	Table:      "table_yearlevels.html",
}

var DefaultYearLevelPathTemplate = "internal/templates/yearlevels/"
