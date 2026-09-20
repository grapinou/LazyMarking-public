package questions

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestPersonalLibraryAndCreation(t *testing.T) {
	t.Chdir("../../..")
	conn := setupQuestionMutationHandlerTest(t)
	for _, table := range []string{"subjects", "themes", "year_levels", "skills", "difficulties"} {
		if _, err := conn.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN name TEXT NOT NULL DEFAULT 'Physique'; INSERT INTO %s(id,user_id,name) VALUES(2,2,'Privé')", table, table)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := conn.Exec(`
 ALTER TABLE points ADD COLUMN point_value INTEGER NOT NULL DEFAULT 1;
 INSERT INTO points(id,user_id,point_value) VALUES(2,2,1);
 CREATE TABLE alt_questions(id INTEGER PRIMARY KEY,question_id INTEGER,content TEXT,user_id INTEGER);
 INSERT INTO questions VALUES(90,2,2,2,2,2,2,'Secret étranger',2);
 `); err != nil {
		t.Fatal(err)
	}
	queries := db.New(conn)
	get := func(path string) string {
		t.Helper()
		rec := authenticatedQuestionRequest(t, http.MethodGet, path, nil, func(w http.ResponseWriter, r *http.Request) { TableQuestionsHandler(w, r, queries) })
		if rec.Code != 200 {
			t.Fatalf("GET %s: %d %s", path, rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}
	empty := get("/dashboard/questions")
	if !strings.Contains(empty, "Créer ma première question") || strings.Contains(empty, "Secret étranger") {
		t.Fatal("empty personal library incorrect")
	}
	form := authenticatedQuestionRequest(t, http.MethodGet, "/dashboard/questions/add", nil, func(w http.ResponseWriter, r *http.Request) { AddFormQuestionsHandler(w, r, queries) })
	if form.Code != 200 || !strings.Contains(form.Body.String(), "Ajouter une question") || !strings.Contains(form.Body.String(), "Retour à Mes questions") {
		t.Fatalf("creation form: %d %s", form.Code, form.Body.String())
	}
	values := url.Values{"content": {"Électricité et énergie"}}
	for _, key := range []string{"subjectID", "themeID", "yearLevelID", "skillID", "difficultyID", "pointID"} {
		values.Set(key, "1")
	}
	created := authenticatedQuestionRequest(t, http.MethodPost, "/dashboard/questions/add", values, func(w http.ResponseWriter, r *http.Request) { AddQuestionsHandler(w, r, queries) })
	if created.Code != http.StatusSeeOther || created.Header().Get("Location") != "/dashboard/questions" {
		t.Fatalf("creation redirect: %v", created)
	}
	if _, err := conn.Exec(`
 INSERT INTO alt_questions VALUES(1,91,'Circuit alternatif',1),(2,91,'Deuxième formulation',1),(3,91,'Variante privée',2),(4,90,'Parent étranger',1);
 `); err != nil {
		t.Fatal(err)
	}
	page := get("/dashboard/questions")
	for _, want := range []string{"Mes questions", "Électricité et énergie", "Matière : Physique", "Niveau : Physique", "Thème : Physique", "2\n", ">Modifier", "/dashboard/questions/add", "Voir les formulations alternatives"} {
		if !strings.Contains(page, want) {
			t.Fatalf("missing %q in library", want)
		}
	}
	for _, forbidden := range []string{"Secret étranger", "Variante privée", "Parent étranger", "Privé"} {
		if strings.Contains(page, forbidden) {
			t.Fatalf("foreign data leaked: %s", forbidden)
		}
	}
	for _, query := range []string{"q=ELECTRICITE", "q=circuit", "subject=Physique", "level=Physique", "theme=Physique", "q=energie&subject=Physique&level=Physique&theme=Physique"} {
		if !strings.Contains(get("/dashboard/questions?"+query), "Électricité et énergie") {
			t.Fatalf("matching search/filter: %s", query)
		}
	}
	for _, query := range []string{"q=inconnue", "q=Secret", "q=privée", "subject=Privé", "level=Privé", "theme=Privé"} {
		page := get("/dashboard/questions?" + query)
		if !strings.Contains(page, "Aucune question ne correspond") || strings.Contains(page, "Créer ma première question") || strings.Contains(page, "Électricité et énergie") {
			t.Fatalf("no results/isolation: %s", query)
		}
	}
}
