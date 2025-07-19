package data

type SubjectRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type SubjectActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultSubjectRoutes = SubjectRoutes{
	AddURL:    "/dashboard/subjects/add",
	EditURL:   "/dashboard/subjects/edit",
	DeleteURL: "/dashboard/subjects/delete",
}

type SubjectPageData struct {
	Routes        DashboardRoutes
	SubjectRoutes SubjectRoutes
	PageTitle     string
	ExtraData     map[string]any
}

type SubjectTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultSubjectTemplateName = SubjectTemplateName{
	AddForm:    "add_form_subject.html",
	EditForm:   "edit_form_subject.html",
	DeleteForm: "delete_form_subject.html",
	Table:      "table_subjects.html",
}

var DefaultSubjectPathTemplate = "internal/templates/subjects/"
