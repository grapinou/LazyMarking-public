package register

import (
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderRegisterPage(w http.ResponseWriter, data data.HomePageData) {
	tmpl := template.Must(template.ParseFiles(
		"internal/templates/home/layout.html",
		"internal/templates/home/register.html",
	))

	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "Can't render layout.html + register.html", http.StatusInternalServerError)
	}
}

func RenderSucessRegister(w http.ResponseWriter, data data.HomePageData) {
	tmpl := template.Must(template.ParseFiles(
		"internal/templates/home/layout.html",
		"internal/templates/home/success.html",
	))

	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "Can't render layout.html + success.html", http.StatusInternalServerError)
	}
}
