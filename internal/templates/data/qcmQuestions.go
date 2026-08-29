package data

import (
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/questionfamilies"
)

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

type QCMQuestionSelectorData struct {
	FilterURL            string
	SubmitURL            string
	TableURL             string
	ResetURL             string
	Subjects             []db.Subject
	Themes               []db.Theme
	YearLevels           []db.YearLevel
	Skills               []db.Skill
	Difficulties         []db.Difficulty
	Points               []db.Point
	QuestionFamilies     []questionfamilies.QuestionFamily
	SelectedSubjectID    int64
	SelectedThemeID      int64
	SelectedYearLevelID  int64
	SelectedSkillID      int64
	SelectedDifficultyID int64
	SelectedPointID      int64
	HasActiveFilters     bool
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
	Selector            QCMQuestionSelectorData
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
