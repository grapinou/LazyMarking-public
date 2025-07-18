package skills

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderSkillPage(w http.ResponseWriter, data data.SkillPageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/skills/skills.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + skills.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func RenderAddSkillForm(w http.ResponseWriter, data data.SkillPageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/skills/addformskills.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + addformskills.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func RenderEditSkillForm(w http.ResponseWriter, data data.SkillPageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/skills/editformskills.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + editformskills.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func RenderDeleteSkillForm(w http.ResponseWriter, data data.SkillPageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/skills/deleteformskills.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + deleteformskills.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
