package login

import (
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderLoginPage(w http.ResponseWriter, data data.HomePageData) {
	tmpl := template.Must(template.ParseFiles(
		"internal/templates/home/layout.html",
		"internal/templates/home/login.html",
	))

	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "Can't render layout.html + login.html", http.StatusInternalServerError)
	}
}
