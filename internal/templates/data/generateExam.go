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
	PageTitle   string
	Routes      DashboardRoutes
	Context     ExamGenerationContext
	Progress    ExamGenerationProgress
	Success     ExamGenerationSuccessData
	Unavailable ExamGenerationUnavailableData
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
	ExamsURL          string
}

func (progress ExamGenerationProgress) Percentage() int64 {
	if progress.TotalStudents <= 0 || progress.ProcessedStudents <= 0 {
		return 0
	}
	if progress.ProcessedStudents >= progress.TotalStudents {
		return 100
	}
	return progress.ProcessedStudents * 100 / progress.TotalStudents
}

type ExamGenerationSuccessData struct {
	Status    string
	CopiesURL string
	ExamsURL  string
}

type ExamGenerationUnavailableData struct {
	ExamsURL string
}

type GenerateExamTemplateName struct {
	ProcessingStudents string
	SuccessProcessing  string
	UnavailablePDF     string
}

var DefaultGenerateExamTemplateName = GenerateExamTemplateName{
	ProcessingStudents: "processing_students.html",
	SuccessProcessing:  "success_processing.html",
	UnavailablePDF:     "unavailable_pdf.html",
}

var DefaultGenerateExamPathTemplate = "internal/templates/generateExam/"
