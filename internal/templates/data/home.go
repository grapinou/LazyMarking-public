package data

type HomeRoutes struct {
	Home                      string
	AboutURL                  string
	LoginURL                  string
	RegisterURL               string
	RegisterSuccessURL        string
	RequestResetPasswordURL   string
	SendEmailResetPasswordURL string
	FormResetPasswordURL      string
	ResetPasswordURL          string
}

var DefaultHomeRoutes = HomeRoutes{
	Home:                      "/",
	AboutURL:                  "/about",
	LoginURL:                  "/login",
	RegisterURL:               "/register",
	RegisterSuccessURL:        "/register/success",
	RequestResetPasswordURL:   "/requestresetpassword",
	SendEmailResetPasswordURL: "/sendemailresetpassword",
	FormResetPasswordURL:      "/formresetpassword",
	ResetPasswordURL:          "/resetpassword",
}

type HomePageData struct {
	RegisterUsername string
	RegisterEmail    string
	RegisterError    string
	RegisterStatus   int
	Routes           HomeRoutes
	PageTitle        string

	ExtraData map[string]any
}
