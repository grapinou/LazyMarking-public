package data

type AltQuestionRoutes struct {
	AddURL        string
	EditURL       string
	DeleteURL     string
	AltAnswersURL string
	AltImageURL   string
	AltPreviewURL string
}

type AltQuestionActionURLs struct {
	EditURL       string
	DeleteURL     string
	AltAnswersURL string
	AltImageURL   string
	AltPreviewURL string
}

var DefaultAltQuestionRoutes = AltQuestionRoutes{
	AddURL:        "/dashboard/questions/altquestions/add",
	EditURL:       "/dashboard/questions/altquestions/edit",
	DeleteURL:     "/dashboard/questions/altquestions/delete",
	AltAnswersURL: "/dashboard/questions/altquestions/altanswers",
	AltImageURL:   "/dashboard/questions/altquestions/altimages",
	AltPreviewURL: "/dashboard/questions/altquestions/altpreview",
}

type AltQuestionPageData struct {
	Routes            DashboardRoutes
	AltQuestionRoutes AltQuestionRoutes
	PageTitle         string
	ExtraData         map[string]any
}

type AltQuestionTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultAltQuestionTemplateName = AltQuestionTemplateName{
	AddForm:    "add_form_alt_question.html",
	EditForm:   "edit_form_alt_question.html",
	DeleteForm: "delete_form_alt_question.html",
	Table:      "table_alt_questions.html",
}

var DefaultAltQuestionPathTemplate = "internal/templates/altquestions/"
