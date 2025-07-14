package data

type DashboardRoutes struct {
	Dashboard    string
	QuestionURL  string
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
	Dashboard:    "/dashboard",
	QuestionURL:  "/questions",
	StudentURL:   "/students",
	QcmURL:       "/qcm",
	ExamURL:      "/exams",
	MarkingURL:   "/marking",
	ResultURL:    "/results",
	DeckURL:      "/flashcards",
	CarrouselURL: "/carrousel",
	LogoutURL:    "/logout",
}

type DashboardPageData struct {
	Routes    DashboardRoutes
	PageTitle string

	ExtraData map[string]any
}
