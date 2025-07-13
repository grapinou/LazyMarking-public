package login

import (
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderLoginPage(w http.ResponseWriter, data data.TemplateLayoutHomeData) {
	tmpl := template.Must(template.ParseFiles(
		"internal/templates/layout_home.html",
		"internal/templates/login.html",
	))

	err := tmpl.ExecuteTemplate(w, "layout_home.html", data)
	if err != nil {
		http.Error(w, "Can't render layout_home.html + login.html", http.StatusInternalServerError)
	}
}
