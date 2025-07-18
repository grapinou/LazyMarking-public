package difficulties

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderDifficultyPage(w http.ResponseWriter, data data.DifficultyPageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/difficulties/difficulties.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + difficulties.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func RenderAddDifficultyForm(w http.ResponseWriter, data data.DifficultyPageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/difficulties/addformdifficulties.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + addformdifficulties.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func RenderEditDifficultyForm(w http.ResponseWriter, data data.DifficultyPageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/difficulties/editformdifficulties.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + editformdifficulty.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func RenderDeleteDifficultyForm(w http.ResponseWriter, data data.DifficultyPageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/difficulties/deleteformdifficulties.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + deleteformdifficulties.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
