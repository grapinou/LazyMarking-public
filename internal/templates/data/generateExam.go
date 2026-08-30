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
	PageTitle string
	Routes    DashboardRoutes
	Context   ExamGenerationContext
	Progress  ExamGenerationProgress
	Success   ExamGenerationSuccessData
}

type ExamGenerationContext struct {
	GenerationID int64
	ExamName     string
	ClassName    string
}

type ExamGenerationProgress struct {
	Status            string
	ProcessedStudents int64
	TotalStudents     int64
	ProgressURL       string
}

type ExamGenerationSuccessData struct {
	Status    string
	CopiesURL string
	ExamsURL  string
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
