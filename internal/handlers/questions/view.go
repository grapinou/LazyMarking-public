package questions

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderQuestionPage(w http.ResponseWriter, data data.DashboardPageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/questions/questions.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + questions.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
