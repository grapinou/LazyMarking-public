package about

import (
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderAboutPage(w http.ResponseWriter, data data.HomePageData) {
	tmpl := template.Must(template.ParseFiles(
		"internal/templates/home/layout.html",
		"internal/templates/home/about.html",
	))

	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "can't render laout.html + about.html", http.StatusInternalServerError)
	}
}
