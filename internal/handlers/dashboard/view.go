package dashboard

import (
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderDashboardPage(w http.ResponseWriter, data data.DashboardPageData) {
	tmpl := template.Must(template.ParseFiles(
		"internal/templates/dashboard/dashboard.html",
		"internal/templates/dashboard/dash_home.html",
	))

	err := tmpl.ExecuteTemplate(w, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + dash_home.html", http.StatusInternalServerError)
	}
}
