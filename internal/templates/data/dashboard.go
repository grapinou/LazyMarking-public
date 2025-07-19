package data

type DashboardRoutes struct {
	DashboardURL    string
	QuestionsURL    string
	AddQuestionURL  string
	SubjectsURL     string
	ThemesURL       string
	YearLevelsURL   string
	SkillsURL       string
	DifficultiesURL string
	PointsURL       string
	StudentURL      string
	QcmURL          string
	ExamURL         string
	MarkingURL      string
	ResultURL       string
	DeckURL         string
	CarrouselURL    string
	LogoutURL       string
}

var DefaultDashboardRoutes = DashboardRoutes{
	DashboardURL:    "/dashboard",
	QuestionsURL:    "/dashboard/questions",
	AddQuestionURL:  "/dashboard/questions/add",
	SubjectsURL:     "/dashboard/subjects",
	ThemesURL:       "/dashboard/themes",
	YearLevelsURL:   "/dashboard/year-levels",
	SkillsURL:       "/dashboard/skills",
	DifficultiesURL: "/dashboard/difficulties",
	PointsURL:       "/dashboard/points",
	StudentURL:      "/dashboard/students",
	QcmURL:          "/dashboard/qcm",
	ExamURL:         "/dashboard/exams",
	MarkingURL:      "/dashboard/marking",
	ResultURL:       "/dashboard/results",
	DeckURL:         "/dashboard/flashcards",
	CarrouselURL:    "/dashboard/carrousel",
	LogoutURL:       "/logout",
}

type DashboardPageData struct {
	Routes    DashboardRoutes
	PageTitle string

	ExtraData map[string]any
}

var (
	DefaultDashboardName = "dashboard.html"
	DefaultDashboarPath  = "internal/templates/dashboard/"
)
