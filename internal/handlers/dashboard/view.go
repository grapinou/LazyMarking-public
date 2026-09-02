package dashboard

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/httpsecurity"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderDashboardPage(w http.ResponseWriter, data data.DashboardPageData) {
	tmpl := template.Must(template.New("dashboard.html").
		Funcs(httpsecurity.TemplateFuncs(w)).
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/dashboard/dashboard.html",
			"internal/templates/dashboard/dash_home.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Can't render dashboard.html + dash_home.html : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
