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

type SubjetTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
}
