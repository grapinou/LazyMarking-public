package dashboard

import (
	"github.com/grapinou/LazyMarking/internal/handlers/about"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVisualGuidesRender(t *testing.T) {
	t.Chdir("../../..")
	public := httptest.NewRecorder()
	about.RenderAboutPage(public, data.HomePageData{Routes: data.DefaultHomeRoutes, PageTitle: "À propos"})
	help := httptest.NewRecorder()
	tools.RenderMergeTemplate(help, data.DashboardPageData{Routes: data.DefaultDashboardRoutes, PageTitle: "Aide"}, data.DefaultDashboarPath, data.DefaultDashboardName, data.DefaultDashboarPath, "help.html")
	for _, w := range []*httptest.ResponseRecorder{public, help} {
		if w.Code != 200 {
			t.Fatalf("render: %d %s", w.Code, w.Body.String())
		}
		for _, text := range []string{"Préparer", "Composer", "Évaluer", "Corriger", "réutiliser", "différence n’est pas garantie"} {
			if !strings.Contains(w.Body.String(), text) {
				t.Fatalf("missing %s", text)
			}
		}
	}
	for _, path := range []string{"/dashboard/questions", "/dashboard/qcm", "/dashboard/exams", "/dashboard/marking"} {
		if !strings.Contains(help.Body.String(), `href="`+path+`"`) {
			t.Fatalf("missing link %s", path)
		}
	}
}

func TestParameterNavigationRendersOnEmptyLists(t *testing.T) {
	t.Chdir("../../..")
	pages := []struct {
		folder, file, next string
		page               any
	}{
		{"subjects", "table_subjects.html", "/dashboard/questions/themes", data.SubjectPageData{Routes: data.DefaultDashboardRoutes}},
		{"themes", "table_themes.html", "/dashboard/questions/year-levels", data.ThemePageData{Routes: data.DefaultDashboardRoutes}},
		{"yearlevels", "table_yearlevels.html", "/dashboard/questions/skills", data.YearLevelPageData{Routes: data.DefaultDashboardRoutes}},
		{"skills", "table_skills.html", "/dashboard/questions/difficulties", data.SkillPageData{Routes: data.DefaultDashboardRoutes}},
		{"difficulties", "table_difficulties.html", "/dashboard/questions/points", data.DifficultyPageData{Routes: data.DefaultDashboardRoutes}},
		{"points", "table_points.html", "/dashboard/questions/add", data.PointPageData{Routes: data.DefaultDashboardRoutes}},
	}
	for _, p := range pages {
		w := httptest.NewRecorder()
		tools.RenderMergeTemplate(w, p.page, data.DefaultDashboarPath, data.DefaultDashboardName, "internal/templates/"+p.folder+"/", p.file)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `href="`+p.next+`"`) || !strings.Contains(w.Body.String(), `aria-label="Parcours des paramètres"`) {
			t.Fatalf("%s: %d %s", p.folder, w.Code, w.Body.String())
		}
	}
}
