package data

type GenerateExamRoutes struct {
	ProcessingStudents string
	MiniQCMLandscape   string
}

var DefaultGenerateExamRoutes = GenerateExamRoutes{
	ProcessingStudents: "/dashboard/exam/ProcessingStudents",
	MiniQCMLandscape:   "/dashboard/exam/user_mini_qcm_landscape",
}

type GenerateExamPageData struct {
	Routes             DashboardRoutes
	GenerateExamRoutes GenerateExamRoutes
	PageTitle          string
	ExtraData          map[string]any
}

type GenerateExamTemplateName struct {
	ProcessingStudents string
}

var DefaultGenerateExamTemplateName = GenerateExamTemplateName{
	ProcessingStudents: "processing_students.html",
}

var DefaultGenerateExamPathTemplate = "internal/templates/generateExam/"
