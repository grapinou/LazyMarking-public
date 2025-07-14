package home

import (
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderHomePage(w http.ResponseWriter, data data.HomePageData) {
	tmpl := template.Must(template.ParseFiles(
		"internal/templates/home/layout.html",
		"internal/templates/home/home.html",
	))

	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "Can't render layout.html + home.html", http.StatusInternalServerError)
	}
}
