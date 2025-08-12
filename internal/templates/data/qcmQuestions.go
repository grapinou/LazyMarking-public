package data

type QCMQuestionRoutes struct {
	AddURL    string
	DeleteURL string
}

type QCMQuestionActionURLs struct {
	DeleteURL string
}

var DefaultQCMQuestionRoutes = QCMQuestionRoutes{
	AddURL:    "/dashboard/QCM/question/add",
	DeleteURL: "/dashboard/QCM/question/delete",
}

type QCMQuestionPageData struct {
	Routes            DashboardRoutes
	QCMQuestionRoutes QCMQuestionRoutes
	PageTitle         string
	ExtraData         map[string]any
}

type QCMQuestionTemplateName struct {
	AddForm    string
	DeleteForm string
	Table      string
}

var DefaultQCMQuestionTemplateName = QCMTemplateName{
	AddForm:    "add_form_qcm_question.html",
	DeleteForm: "delete_form_qcm_question.html",
	Table:      "table_qcmquestion.html",
}

var DefaultQCMQuestionPathTemplate = "internal/templates/qcmquestions/"
