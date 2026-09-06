package register

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/httpsecurity"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderRegisterPage(w http.ResponseWriter, data data.HomePageData) {
	tmpl := template.Must(template.New("layout.html").
		Funcs(httpsecurity.TemplateFuncs(w)).
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/home/layout.html",
			"internal/templates/home/register.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "layout.html", data)
	if err != nil {
		http.Error(w, "Impossible d’afficher cette page.", http.StatusInternalServerError)
		return
	}
	status := data.RegisterStatus
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w)
}

func RenderSucessRegister(w http.ResponseWriter, data data.HomePageData) {
	tmpl := template.Must(template.New("layout.html").
		Funcs(httpsecurity.TemplateFuncs(w)).
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/home/layout.html",
			"internal/templates/home/success.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "layout.html", data)
	if err != nil {
		http.Error(w, "Impossible d’afficher cette page.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
