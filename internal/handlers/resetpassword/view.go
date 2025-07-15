package resetpassword

import (
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderShowRequestForm(w http.ResponseWriter, data data.HomePageData) {
	tmpl := template.Must(template.ParseFiles(
		"internal/templates/home/layout.html",
		"internal/templates/home/showrequestresetpasswordform.html",
	))

	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "Can't render layout.html + showrequestresetpasswordform.html", http.StatusInternalServerError)
	}
}

func RenderShowResetForm(w http.ResponseWriter, data data.HomePageData) {
	tmpl := template.Must(template.ParseFiles(
		"internal/templates/home/layout.html",
		"internal/templates/home/showresetpasswordform.html",
	))

	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "Can't render layout.html + showresetpasswordform.html", http.StatusInternalServerError)
	}
}
