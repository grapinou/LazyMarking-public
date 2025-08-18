package data

type ExamRoutes struct {
	YearsURL   string
	PeriodsURL string
	AddURL     string
	EditURL    string
	DeleteURL  string
}

type ExamActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultExamRoutes = ExamRoutes{
	YearsURL:   "/dashboard/exams/years",
	PeriodsURL: "/dashboard/exams/periods",
	AddURL:     "/dashboard/exams/add",
	EditURL:    "/dashboard/exams/edit",
	DeleteURL:  "/dashboard/exams/delete",
}

type ExamPageData struct {
	Routes     DashboardRoutes
	ExamRoutes ExamRoutes
	PageTitle  string
	ExtraData  map[string]any
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
