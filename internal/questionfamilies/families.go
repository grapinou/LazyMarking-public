package questionfamilies

import "sort"

// Question is the application-facing representation of a main question.
// Metadata is populated for views, such as the QCM selector, that need it.
type Question struct {
	ID             int64
	Content        string
	SubjectName    string
	ThemeName      string
	YearLevelName  string
	SkillName      string
	DifficultyName string
	PointValue     int64
	Selectable     bool
}

type Variant struct {
	ID         int64
	QuestionID int64
	Content    string
}

type QuestionFamily struct {
	Main     Question
	Variants []Variant
}

// Build groups variants under their main question. It deliberately does not
// implement ownership checks: those belong to the SQL reads that feed it.
func Build(questions []Question, variants []Variant) []QuestionFamily {
	families := make([]QuestionFamily, 0, len(questions))
	byQuestionID := make(map[int64]int, len(questions))
	for _, question := range questions {
		byQuestionID[question.ID] = len(families)
		families = append(families, QuestionFamily{
			Main:     question,
			Variants: make([]Variant, 0),
		})
	}

	ownedVariants := append([]Variant(nil), variants...)
	sort.SliceStable(ownedVariants, func(i, j int) bool {
		if ownedVariants[i].QuestionID == ownedVariants[j].QuestionID {
			return ownedVariants[i].ID < ownedVariants[j].ID
		}
		return ownedVariants[i].QuestionID < ownedVariants[j].QuestionID
	})
	for _, variant := range ownedVariants {
		if familyIndex, ok := byQuestionID[variant.QuestionID]; ok {
			families[familyIndex].Variants = append(families[familyIndex].Variants, variant)
		}
	}

	return families
}
