package about

import (
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderAboutPage(w http.ResponseWriter, data data.TemplateLayoutHomeData) {

	tmpl := template.Must(template.ParseFiles(
		"internal/templates/layout_home.html",
		"internal/templates/about.html",
	))

	err := tmpl.ExecuteTemplate(w, "layout_home.html", data)
	if err != nil {
		http.Error(w, "can't render laout_home.html + about.html", http.StatusInternalServerError)
	}
}