package data

type PeriodRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type PeriodActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultPeriodRoutes = PeriodRoutes{
	AddURL:    "/dashboard/exams/periods/add",
	EditURL:   "/dashboard/exams/periods/edit",
	DeleteURL: "/dashboard/exams/periods/delete",
}

type PeriodPageData struct {
	Routes       DashboardRoutes
	PeriodRoutes PeriodRoutes
	PageTitle    string
	ExtraData    map[string]any
}

type PeriodTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultPeriodTemplateName = PeriodTemplateName{
	AddForm:    "add_form_period.html",
	EditForm:   "edit_form_period.html",
	DeleteForm: "delete_form_period.html",
	Table:      "table_periods.html",
}

var DefaultPeriodPathTemplate = "internal/templates/periods/"
