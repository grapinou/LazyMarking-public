package data

type AltAnswerRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type AltAnswerActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultAltAnswerRoutes = AnswerRoutes{
	AddURL:    "/dashboard/questions/altquestions/altanswers/add",
	EditURL:   "/dashboard/questions/altquestions/altanswers/edit",
	DeleteURL: "/dashboard/questions/altquestions/altanswers/delete",
}

type AltAnswerPageData struct {
	Routes          DashboardRoutes
	AltAnswerRoutes AltAnswerRoutes
	PageTitle       string
	ExtraData       map[string]any
}

type AltAnswerTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultAltAnswerTemplateName = AltAnswerTemplateName{
	AddForm:    "add_form_alt_answer.html",
	EditForm:   "edit_form_alt_answer.html",
	DeleteForm: "delete_form_alt_answer.html",
	Table:      "table_alt_answers.html",
}

var DefaultAltAnswerPathTemplate = "internal/templates/altanswers/"
