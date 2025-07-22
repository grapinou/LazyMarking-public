package data

type AnswerRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type AnswerActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultAnswerRoutes = AnswerRoutes{
	AddURL:    "/dashboard/questions/answers/add",
	EditURL:   "/dashboard/questions/answers/edit",
	DeleteURL: "/dashboard/questions/answers/delete",
}

type AnswerPageData struct {
	Routes       DashboardRoutes
	AnswerRoutes AnswerRoutes
	PageTitle    string
	ExtraData    map[string]any
}

type AnswerTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultAnswerTemplateName = AnswerTemplateName{
	AddForm:    "add_form_answer.html",
	EditForm:   "edit_form_answer.html",
	DeleteForm: "delete_form_answer.html",
	Table:      "table_answers.html",
}

var DefaultAnswerPathTemplate = "internal/templates/answers/"
