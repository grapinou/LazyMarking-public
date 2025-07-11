package about

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func AboutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := data.TemplateLayoutHomeData{
		Home:              "/",
		AboutURL: "/about",
		LoginURL:          "/login",
		RegisterURL:       "/register",
		ForgotPasswordURL: "/forgot-password",
		PageTitle:         "About",
	}

	RenderAboutPage(w, data)


}