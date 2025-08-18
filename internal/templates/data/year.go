package data

type YearRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type YearActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultYearRoutes = YearRoutes{
	AddURL:    "/dashboard/exams/years/add",
	EditURL:   "/dashboard/exams/years/edit",
	DeleteURL: "/dashboard/exams/years/delete",
}

type YearPageData struct {
	Routes     DashboardRoutes
	YearRoutes YearRoutes
	PageTitle  string
	ExtraData  map[string]any
}

type YearTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultYearTemplateName = YearTemplateName{
	AddForm:    "add_form_year.html",
	EditForm:   "edit_form_year.html",
	DeleteForm: "delete_form_year.html",
	Table:      "table_years.html",
}

var DefaultYearPathTemplate = "internal/templates/years/"
