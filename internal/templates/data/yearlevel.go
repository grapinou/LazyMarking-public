package data

type YearLevelRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type YearYevelActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultYearLevelRoutes = YearLevelRoutes{
	AddURL:    "/dashboard/yearlevels/add",
	EditURL:   "/dashboard/yearlevels/edit",
	DeleteURL: "/dashboard/yearlevels/delete",
}

type YearLevelPageData struct {
	Routes          DashboardRoutes
	YearLevelRoutes YearLevelRoutes
	PageTitle       string
	ExtraData       map[string]any
}
