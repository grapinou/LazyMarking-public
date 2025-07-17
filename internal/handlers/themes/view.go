package themes

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderThemePage(w http.ResponseWriter, data data.ThemePageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/themes/themes.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + themes.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func RenderAddThemeForm(w http.ResponseWriter, data data.ThemePageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/themes/addformthemes.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + addformthemes.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func RenderEditThemeForm(w http.ResponseWriter, data data.ThemePageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/themes/editformthemes.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + editformthemes.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func RenderDeleteThemeForm(w http.ResponseWriter, data data.ThemePageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/themes/deleteformthemes.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + deleteformthemes.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
