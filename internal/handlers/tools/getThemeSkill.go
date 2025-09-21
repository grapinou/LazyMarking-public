package tools

import (
	"fmt"

	"github.com/grapinou/LazyMarking/internal/config"
)

func GetThemeSkill(qcm config.QCM, questionsState []config.QuestionMark) (map[int64]config.CounterTag, map[string]config.CounterTag) {
	// obtenir les skills et les themes skills
	skill := make(map[int64]config.CounterTag)
	themeSkill := make(map[string]config.CounterTag)

	for i, question := range qcm.Questions {
		s := question.Tags.Skill
		t := question.Tags.Theme

		skillID := s.ID
		themeID := t.ID

		cSkill := skill[skillID] // récupère (ou zéro value)
		// on met à jour
		cSkill.Name = s.Name
		cSkill.Score += questionsState[i].Score
		cSkill.Total += questionsState[i].Total
		// on réinjecte dans la map
		skill[skillID] = cSkill

		key := fmt.Sprintf("%d-%d", themeID, skillID)
		cThemeSkill := themeSkill[key]
		cThemeSkill.Name = t.Name + "-" + s.Name
		cThemeSkill.Score += questionsState[i].Score
		cThemeSkill.Total += questionsState[i].Total
		themeSkill[key] = cThemeSkill

	}
	return skill, themeSkill
}
