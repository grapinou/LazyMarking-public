package data

type HomeRoutes struct {
	Home              string
	AboutURL          string
	LoginURL          string
	RegisterURL       string
	ForgotPasswordURL string
	PageTitle         string
}

var DefaultHomeRoutes = HomeRoutes{
	Home:              "/",
	AboutURL:          "/about",
	LoginURL:          "/login",
	RegisterURL:       "/register",
	ForgotPasswordURL: "/forgot-password",
}

type HomePageData struct {
	Routes    HomeRoutes
	PageTitle string

	ExtraData map[string]any
}
