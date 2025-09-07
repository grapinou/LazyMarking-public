package data

type GenerateExamRoutes struct {
	ProcessingStudents string
	SuccessProcessing  string
	PdfExam            string
	MiniQCMLandscape   string
}

var DefaultGenerateExamRoutes = GenerateExamRoutes{
	ProcessingStudents: "/dashboard/exam/ProcessingStudents",
	SuccessProcessing:  "/dashboard/exam/SuccessProcessing",
	PdfExam:            "/dashboard/exam/pdf_exam",
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
	SuccessProcessing  string
}

var DefaultGenerateExamTemplateName = GenerateExamTemplateName{
	ProcessingStudents: "processing_students.html",
	SuccessProcessing:  "success_processing.html",
}

var DefaultGenerateExamPathTemplate = "internal/templates/generateExam/"
