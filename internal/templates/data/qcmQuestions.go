package data

type QCMQuestionRoutes struct {
	AddURL      string
	DeleteURL   string
	MoveUpURL   string
	MoveDownURL string
}

type QCMQuestionItem struct {
	QCMQuestionID int64
	Position      int64
	Content       string
	IsFirst       bool
	IsLast        bool
	MoveUpURL     string
	MoveDownURL   string
	DeleteURL     string
}

var DefaultQCMQuestionRoutes = QCMQuestionRoutes{
	AddURL:      "/dashboard/qcm/qcmquestion/add",
	DeleteURL:   "/dashboard/qcm/qcmquestion/delete",
	MoveUpURL:   "/dashboard/qcm/qcmquestion/move-up",
	MoveDownURL: "/dashboard/qcm/qcmquestion/move-down",
}

type QCMQuestionPageData struct {
	Routes              DashboardRoutes
	QCMQuestionRoutes   QCMQuestionRoutes
	QCMContext          QCMContext
	QCMQuestions        []QCMQuestionItem
	AddQuestionsURL     string
	PreviewURL          string
	PreviewLandscapeURL string
	PageTitle           string
	ExtraData           map[string]any
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
