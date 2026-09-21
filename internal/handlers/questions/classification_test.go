package questions

import (
	"context"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/difficulties"
	"github.com/grapinou/LazyMarking/internal/handlers/skills"
	"github.com/grapinou/LazyMarking/internal/handlers/subjects"
	"github.com/grapinou/LazyMarking/internal/handlers/themes"
	"github.com/grapinou/LazyMarking/internal/handlers/yearlevels"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestManualClassificationCreationReusesLabelsAndShowsAdvice(t *testing.T) {
	t.Chdir("../../..")
	conn, _, err := db.OpenMigratedDB(context.Background(), filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`INSERT INTO users(id,username,email,hashpassword) VALUES(1,'Bob','b@example.test','h'),(2,'Alice','a@example.test','h')`); err != nil {
		t.Fatal(err)
	}
	type handler func(http.ResponseWriter, *http.Request, *db.Queries)
	for _, tc := range []struct {
		table, field, existing, attempt, group, listURL, editParam string
		add, addForm, editForm, list                               handler
	}{
		{"subjects", "subject", "Physique Chimie", " physique  CHIMIE ", "matières", data.DefaultQuestionRoutes.SubjectsURL, "subject_id", subjects.AddSubjectHandler, subjects.AddFormSubjectHandler, subjects.EditFormSubjectHandler, subjects.TableSubjectsHandler},
		{"year_levels", "yearlevel", "Seconde", "seconde", "niveaux", data.DefaultQuestionRoutes.YearLevelsURL, "yearlevel_id", yearlevels.AddYearLevelHandler, yearlevels.AddFormYearLevelHandler, yearlevels.EditFormYearLevelHandler, yearlevels.TableYearLevelsHandler},
		{"themes", "theme", "Electricite", "ÉLECTRICITÉ", "thèmes", data.DefaultQuestionRoutes.ThemesURL, "theme_id", themes.AddThemeHandler, themes.AddFormThemeHandler, themes.EditFormThemeHandler, themes.TableThemesHandler},
		{"skills", "skill", "Évaluer", "e\u0301valuer", "compétences", data.DefaultQuestionRoutes.SkillsURL, "skill_id", skills.AddSkillHandler, skills.AddFormSkillHandler, skills.EditFormSkillHandler, skills.TableSkillsHandler},
		{"difficulties", "difficulty", "Facile", " FACILE ", "difficultés", data.DefaultQuestionRoutes.DifficultiesURL, "difficulty_id", difficulties.AddDifficultyHandler, difficulties.AddFormDifficultyHandler, difficulties.EditFormDifficultyHandler, difficulties.TableDifficultiesHandler},
	} {
		t.Run(tc.table, func(t *testing.T) {
			if _, err := conn.Exec("INSERT INTO "+tc.table+"(id,name,user_id) VALUES(1,?,1),(2,'Secret étranger',2)", tc.existing); err != nil {
				t.Fatal(err)
			}
			q := db.New(conn)
			serve := func(method, path string, form url.Values, h handler) (int, string, string) {
				t.Helper()
				r := authenticatedQuestionRequest(t, method, path, form, func(w http.ResponseWriter, r *http.Request) { h(w, r, q) })
				return r.Code, r.Body.String(), r.Header().Get("Location")
			}
			status, body, location := serve("POST", "/add", url.Values{tc.field: {tc.attempt}}, tc.add)
			if status != 303 || location != tc.listURL+"?existing=1" {
				t.Fatalf("creation: %d %s %s", status, body, location)
			}
			status, body, _ = serve("GET", location, nil, tc.list)
			want := "« " + tc.existing + " » existe déjà dans vos " + tc.group
			if status != 200 || !strings.Contains(body, want) || !strings.Contains(body, "Conseil :") {
				t.Fatalf("duplicate notice: %d %s", status, body)
			}
			var count int
			var stored string
			if err := conn.QueryRow("SELECT COUNT(*) FROM " + tc.table + " WHERE user_id=1").Scan(&count); err != nil || count != 1 {
				t.Fatalf("duplicate created: %d %v", count, err)
			}
			if err := conn.QueryRow("SELECT name FROM " + tc.table + " WHERE id=1").Scan(&stored); err != nil || stored != tc.existing {
				t.Fatalf("label changed: %q %v", stored, err)
			}
			for _, page := range []struct {
				h    handler
				path string
			}{{tc.addForm, "/add"}, {tc.editForm, "/edit?" + tc.editParam + "=1"}} {
				status, body, _ := serve("GET", page.path, nil, page.h)
				if status != 200 || !strings.Contains(body, "Conseil :") || !strings.Contains(body, "Vos appellations restent libres.") {
					t.Fatalf("advice: %d %s", status, body)
				}
			}
			status, body, _ = serve("GET", tc.listURL+"?existing=2", nil, tc.list)
			if status != 200 || strings.Contains(body, "Secret étranger") || strings.Contains(body, "existe déjà") {
				t.Fatal("foreign duplicate notice exposed")
			}
			status, _, location = serve("POST", "/add", url.Values{tc.field: {"2nde"}}, tc.add)
			if status != 303 || location != tc.listURL {
				t.Fatalf("new label blocked: %d %s", status, location)
			}
		})
	}
}
