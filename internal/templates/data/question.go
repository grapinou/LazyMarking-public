package data

type QuestionRoutes struct {
	SubjectsURL     string
	ThemesURL       string
	YearLevelsURL   string
	SkillsURL       string
	DifficultiesURL string
	PointsURL       string
	AddURL          string
	EditURL         string
	DeleteURL       string
}

type QuestionActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultQuestionRoutes = QuestionRoutes{
	SubjectsURL:     "/dashboard/questions/subjects",
	ThemesURL:       "/dashboard/questions/themes",
	YearLevelsURL:   "/dashboard/questions/year-levels",
	SkillsURL:       "/dashboard/questions/skills",
	DifficultiesURL: "/dashboard/questions/difficulties",
	PointsURL:       "/dashboard/questions/points",
	AddURL:          "/dashboard/questions/add",
	EditURL:         "/dashboard/questions/edit",
	DeleteURL:       "/dashboard/questions/delete",
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
