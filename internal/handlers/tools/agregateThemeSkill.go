package tools

import "github.com/grapinou/LazyMarking/internal/config"

func AgregateThemeSkill(marks []config.MarkExam) (map[int64]config.CounterTag, map[string]config.CounterTag) {
	globalSkills := make(map[int64]config.CounterTag)
	globalThemeSkills := make(map[string]config.CounterTag)

	for _, m := range marks {
		for id, ct := range m.Skill {
			if agg, ok := globalSkills[id]; ok {
				agg.Total += ct.Total
				agg.Score += ct.Score
				if agg.Name == "" {
					agg.Name = ct.Name
				}
				globalSkills[id] = agg
			} else {
				// copy to avoid aliasing (pas strictement nécessaire ici mais plus sûr)
				globalSkills[id] = config.CounterTag{
					Name:  ct.Name,
					Score: ct.Score,
					Total: ct.Total,
				}
			}
		}

		for k, ct := range m.ThemeSkill {
			if agg, ok := globalThemeSkills[k]; ok {
				agg.Total += ct.Total
				agg.Score += ct.Score
				if agg.Name == "" {
					agg.Name = ct.Name
				}
				globalThemeSkills[k] = agg
			} else {
				globalThemeSkills[k] = config.CounterTag{
					Name:  ct.Name,
					Score: ct.Score,
					Total: ct.Total,
				}
			}
		}
	}

	return globalSkills, globalThemeSkills
}
