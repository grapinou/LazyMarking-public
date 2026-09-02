package tools

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/httpsecurity"
)

// RenderMergeTemplate combine deux templates HTML et les exécute avec les données fournies.
// Le paramètre `data` doit être un struct provenant du package `data` (ex: data.SkillPageData).
// En cas d'erreur de rendu, la fonction envoie une réponse HTTP 500 avec le message d'erreur.
// Aucun résultat n'est retourné par la fonction.
func RenderMergeTemplate(w http.ResponseWriter, data any, rootPathTemplate, rootName, pathTemplate, nameTemplate string) {
	tmpl := template.Must(template.New(rootName).
		Funcs(httpsecurity.TemplateFuncs(w)).
		Option("missingkey=error").
		ParseFiles(
			rootPathTemplate+rootName,
			pathTemplate+nameTemplate,
		))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, rootName, data)
	if err != nil {
		http.Error(w, "Can't render "+rootName+" "+nameTemplate+" : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
