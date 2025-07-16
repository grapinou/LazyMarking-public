package about

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderAboutPage(w http.ResponseWriter, data data.HomePageData) {
	tmpl := template.Must(template.New("layout.html").
		Option("missingkey=error").
		ParseFiles(
			"internal/templates/home/layout.html",
			"internal/templates/home/about.html",
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, "layout.html", data)
	if err != nil {
		http.Error(w, "can't render laout.html + about.html", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
