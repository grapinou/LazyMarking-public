package home

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := data.TemplateLayoutHomeData{
		Home:              "/",
		AboutURL:          "/about",
		LoginURL:          "/login",
		RegisterURL:       "/register",
		ForgotPasswordURL: "/forgot-password",
		PageTitle:         "Home",
	}

	RenderHomePage(w, data)
}
