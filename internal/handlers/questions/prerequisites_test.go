package questions

import (
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"testing"
)

func TestQuestionPrerequisitesProgressAndReady(t *testing.T) {
	features := map[string]any{"subjects": []db.Subject{}, "themes": []db.Theme{}, "yearLevels": []db.YearLevel{}, "skills": []db.Skill{}, "difficulties": []db.Difficulty{}, "points": []db.Point{}}
	keys := []string{"subjects", "themes", "yearLevels", "skills", "difficulties", "points"}
	values := []any{[]db.Subject{{}}, []db.Theme{{}}, []db.YearLevel{{}}, []db.Skill{{}}, []db.Difficulty{{}}, []db.Point{{}}}
	urls := []string{data.DefaultSubjectRoutes.AddURL, data.DefaultThemeRoutes.AddURL, data.DefaultYearLevelRoutes.AddURL, data.DefaultSkillRoutes.AddURL, data.DefaultDifficultyRoutes.AddURL, data.DefaultPointRoutes.AddURL, data.DefaultQuestionRoutes.AddURL}
	for i := 0; i <= 6; i++ {
		prerequisites, next := questionPrerequisites(features)
		if next != urls[i] || len(prerequisites) != 6 {
			t.Fatalf("step %d: next=%s", i, next)
		}
		for j, p := range prerequisites {
			if p.Available != (j < i) {
				t.Fatalf("step %d prerequisite %d", i, j)
			}
		}
		if i < 6 {
			features[keys[i]] = values[i]
		}
	}
}
