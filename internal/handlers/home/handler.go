package home

import (
	"net/http"
)

type TemplateHomeData struct {
	Home              string
	LoginURL          string
	RegisterURL       string
	ForgotPasswordURL string
	PageTitle         string
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := TemplateHomeData{
		Home:              "/",
		LoginURL:          "/login",
		RegisterURL:       "/register",
		ForgotPasswordURL: "/forgot-password",
		PageTitle:         "Home",
	}

	RenderHomePage(w, data)
}
