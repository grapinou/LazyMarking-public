package resetpassword

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderShowRequestForm(w http.ResponseWriter, data data.HomePageData) {
	tmpl := template.Must(template.New("layout.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/home/layout.html",
			"internal/templates/home/showrequestresetpasswordform.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "layout.html", data)
	if err != nil {
		http.Error(w, "Can't render layout.html + showrequestresetpasswordform.html", http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func RenderShowResetForm(w http.ResponseWriter, data data.HomePageData) {
	tmpl := template.Must(template.New("layout.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/home/layout.html",
			"internal/templates/home/showresetpasswordform.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "layout.html", data)
	if err != nil {
		http.Error(w, "Can't render layout.html + showresetpasswordform.html", http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
