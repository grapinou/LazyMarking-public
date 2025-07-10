package home

import (
	"html/template"
	"net/http"
)

func RenderHomePage(w http.ResponseWriter, data TemplateHomeData) {

	tmpl := template.Must(template.ParseFiles(
		"templates/layout_home.html",
		"templates/home.html",
	))

	err := tmpl.ExecuteTemplate(w, "layout_home.html", data)
	if err != nil {
		http.Error(w, "Can't render layout_home.html", http.StatusInternalServerError)
	}

}
