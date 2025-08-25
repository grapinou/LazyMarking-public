package data

type QCMRoutes struct {
	AddURL              string
	EditURL             string
	DeleteURL           string
	AddQuestionURL      string
	PreviewURL          string
	PreviewLandscapeURL string
}

type QCMActionURLs struct {
	EditURL             string
	DeleteURL           string
	AddQuestionURL      string
	PreviewURL          string
	PreviewLandscapeURL string
}

var DefaultQCMRoutes = QCMRoutes{
	AddURL:              "/dashboard/qcm/add",
	EditURL:             "/dashboard/qcm/edit",
	DeleteURL:           "/dashboard/qcm/delete",
	AddQuestionURL:      "/dashboard/qcm/qcmquestion",
	PreviewURL:          "/dashboard/qcm/previewqcm",
	PreviewLandscapeURL: "/dashboard/qcm/previewqcmlandscape",
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
