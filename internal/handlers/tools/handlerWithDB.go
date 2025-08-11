package tools

import (
	"database/sql"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
)

// HandlerWithDB est un adaptateur pour les handlers HTTP nécessitant un accès à la base de données.
//
// Elle prend une fonction de handler ayant la signature typique suivante :
//
//	func(w http.ResponseWriter, r *http.Request, queries *db.Queries)
//
// et la transforme en http.Handler compatible avec http.ServeMux.
//
// Cela permet d'injecter l'instance *db.Queries dans les handlers sans devoir créer une closure manuellement.
//
// Exemple d'utilisation :
//
//	mux.Handle("/dashboard/skills", CheckAuth(HandlerWithDB(SkillsHandler, queries)))
func HandlerWithDB(fn func(http.ResponseWriter, *http.Request, *db.Queries), queries *db.Queries) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, queries)
	})
}

func HandlerWithDBAndConn(fn func(http.ResponseWriter, *http.Request, *db.Queries, *sql.DB), queries *db.Queries, conn *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, queries, conn)
	})
}
