package data

type PointRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type PointActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultPointRoutes = PointRoutes{
	AddURL:    "points/add",
	EditURL:   "points/edit",
	DeleteURL: "points/delete",
}

type PointPageData struct {
	Routes      DashboardRoutes
	PointRoutes PointRoutes
	PageTitle   string
	ExtraData   map[string]any
}
