package data

type PointRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type PointActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultPointRoutes = PointRoutes{
	AddURL:    "/dashboard/points/add",
	EditURL:   "/dashboard/points/edit",
	DeleteURL: "/dashboard/points/delete",
}

type PointPageData struct {
	Routes      DashboardRoutes
	PointRoutes PointRoutes
	PageTitle   string
	ExtraData   map[string]any
}

type PointTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultPointTemplateName = PointTemplateName{
	AddForm:    "add_form_point.html",
	EditForm:   "edit_form_point.html",
	DeleteForm: "delete_form_point.html",
	Table:      "table_points.html",
}

var DefaultPointPathTemplate = "internal/templates/points/"
