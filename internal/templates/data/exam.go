package data

import "github.com/grapinou/LazyMarking/internal/db"

type ExamRoutes struct {
	YearsURL        string
	PeriodsURL      string
	AddURL          string
	EditURL         string
	DeleteURL       string
	GenerateExamPdf string
	GenerateMiniPdf string
}

type ExamListItem struct {
	ID         int64
	Name       string
	QCMName    string
	ClassName  string
	YearName   string
	PeriodName string

	EditURL     string
	DeleteURL   string
	GenerateURL string
	MiniURL     string
}

type ExamContext struct {
	ID   int64
	Name string
}

type ExamFormData struct {
	QCMs    []db.GetAllQCMRow
	Classes []db.ClassCode
	Years   []db.Year
	Periods []db.Period
	Name    string

	SelectedQCMID    int64
	SelectedClassID  int64
	SelectedYearID   int64
	SelectedPeriodID int64
}

var DefaultExamRoutes = ExamRoutes{
	YearsURL:        "/dashboard/exams/years",
	PeriodsURL:      "/dashboard/exams/periods",
	AddURL:          "/dashboard/exams/add",
	EditURL:         "/dashboard/exams/edit",
	DeleteURL:       "/dashboard/exams/delete",
	GenerateExamPdf: "/dashboard/exams/generate",
	GenerateMiniPdf: "/dashboard/exams/generatemini",
}

type ExamPageData struct {
	Routes     DashboardRoutes
	ExamRoutes ExamRoutes
	PageTitle  string
	Items      []ExamListItem
	Exam       ExamContext
	Form       ExamFormData
	CancelURL  string
}

type ExamTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultExamTemplateName = ExamTemplateName{
	AddForm:    "add_form_exam.html",
	EditForm:   "edit_form_exam.html",
	DeleteForm: "delete_form_exam.html",
	Table:      "table_exams.html",
}

var DefaultExamPathTemplate = "internal/templates/exams/"
