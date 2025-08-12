package data

type QCMRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type QCMActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultQCMRoutes = QCMRoutes{
	AddURL:    "/dashboard/QCM/add",
	EditURL:   "/dashboard/QCM/edit",
	DeleteURL: "/dashboard/QCM/delete",
}

type QCMPageData struct {
	Routes    DashboardRoutes
	QCMRoutes QCMRoutes
	PageTitle string
	ExtraData map[string]any
}

type QCMTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultQCMTemplateName = QCMTemplateName{
	AddForm:    "add_form_qcm.html",
	EditForm:   "edit_form_qcm.html",
	DeleteForm: "delete_form_qcm.html",
	Table:      "table_qcm.html",
}

var DefaultQCMPathTemplate = "internal/templates/qcm/"
