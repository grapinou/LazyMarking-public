package tools

import (
	"log"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
)

func GetAllFeaturesQuestion(r *http.Request, userID int64, queries *db.Queries) (map[string]any, bool) {
	features := make(map[string]any)
	ok := false

	subjects, err := queries.GetAllSubjects(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQuestionsHandler : GetAllSubject DB error: %v", err)
		return features, ok
	}

	themes, err := queries.GetAllThemes(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQuestionsHandler : GetAllTheme DB error: %v", err)
		return features, ok
	}

	yearLevels, err := queries.GetAllYearLevels(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQuestionsHandler : GetAllYearLevels DB error: %v", err)
		return features, ok
	}

	skills, err := queries.GetAllSkills(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQuestionsHandler : GetAllSkills DB error: %v", err)
		return features, ok
	}

	difficulties, err := queries.GetAllDifficulties(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQuestionsHandler : GetAllDifficulties DB error: %v", err)
		return features, ok
	}
	points, err := queries.GetAllPoints(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQuestionsHandler : GetAllPoints DB error: %v", err)
		return features, ok
	}

	features["subjects"] = subjects
	features["themes"] = themes
	features["yearLevels"] = yearLevels
	features["skills"] = skills
	features["difficulties"] = difficulties
	features["points"] = points

	ok = true
	return features, ok
}
