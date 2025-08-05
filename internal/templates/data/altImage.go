package data

type AltImageRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

var DefaultAltImageRoutes = AltImageRoutes{
	AddURL:    "/dashboard/altquestions/altimages/add",
	EditURL:   "/dashboard/altquestions/altimages/edit",
	DeleteURL: "/dashboard/altquestions/altimages/delete",
}

type AltImagePageData struct {
	Routes         DashboardRoutes
	AltImageRoutes AltImageRoutes
	PageTitle      string
	ExtraData      map[string]any
}

type AltImageTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultAltImageTemplateName = AltImageTemplateName{
	AddForm:    "add_form_alt_image.html",
	EditForm:   "edit_form_alt_image.html",
	DeleteForm: "delete_form_alt_image.html",
	Table:      "table_alt_image.html",
}

var DefaultAltImagePathTemplate = "internal/templates/altimages/"
