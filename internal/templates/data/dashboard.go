package data

type DashboardRoutes struct {
	DashboardURL string
	QuestionsURL string
	StudentURL   string
	QcmURL       string
	ExamURL      string
	MarkingURL   string
	ResultURL    string
	DeckURL      string
	CarrouselURL string
	LogoutURL    string
}

var DefaultDashboardRoutes = DashboardRoutes{
	DashboardURL: "/dashboard",
	QuestionsURL: "/dashboard/questions",
	StudentURL:   "/dashboard/students",
	QcmURL:       "/dashboard/qcm",
	ExamURL:      "/dashboard/exams",
	MarkingURL:   "/dashboard/marking",
	ResultURL:    "/dashboard/results",
	DeckURL:      "/dashboard/flashcards",
	CarrouselURL: "/dashboard/carrousel",
	LogoutURL:    "/logout",
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
