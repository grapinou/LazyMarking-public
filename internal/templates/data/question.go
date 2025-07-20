package data

type QuestionRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type QuestionActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultQuestionRoutes = QuestionRoutes{
	AddURL:    "/dashboard/questions/add",
	EditURL:   "/dashboard/questions/edit",
	DeleteURL: "/dashboard/questions/delete",
}

type QuestionPageData struct {
	Routes         DashboardRoutes
	QuestionRoutes QuestionRoutes
	PageTitle      string
	ExtraData      map[string]any
}

type QuestionTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultQuestionTemplateName = QuestionTemplateName{
	AddForm:    "add_form_question.html",
	EditForm:   "edit_form_question.html",
	DeleteForm: "delete_form_question.html",
	Table:      "table_questions.html",
}

var DefaultQuestionPathTemplate = "internal/templates/questions/"
