package difficulties

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TableDifficultiesHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	difficultiesDB, err := queries.GetAllDifficulties(r.Context(), userID)
	if err != nil {
		log.Printf("From TableDifficultiesHandler : GetAllDiffulties DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	noDifficulties := true
	if len(difficultiesDB) > 0 {
		noDifficulties = false
	}

	var actionsURLParameters []data.DifficultyActionURLs
	if !noDifficulties {
		for _, difficulty := range difficultiesDB {
			editURL := data.DefaultDifficultyRoutes.EditURL + "?difficulty_id=" + url.QueryEscape(strconv.FormatInt(difficulty.ID, 10))
			deleteURL := data.DefaultDifficultyRoutes.DeleteURL + "?difficulty_id=" + url.QueryEscape(strconv.FormatInt(difficulty.ID, 10))

			actionsURLParameters = append(actionsURLParameters, data.DifficultyActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.DifficultyPageData{
		Routes:           data.DefaultDashboardRoutes,
		DifficultyRoutes: data.DefaultDifficultyRoutes,
		PageTitle:        "difficulties",
		ExtraData: map[string]any{
			"NoDifficulties": noDifficulties,
			"Action":         actionsURLParameters,
			"Difficulties":   difficultiesDB,
		},
	}

	RenderTableDifficultyPage(w, dataPage)
}

func AddFormDifficultyHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	dataPage := data.DifficultyPageData{
		Routes:           data.DefaultDashboardRoutes,
		DifficultyRoutes: data.DefaultDifficultyRoutes,
		PageTitle:        "add difficulty",
	}
	RenderAddFormDifficulty(w, dataPage)
}

func AddDifficultyHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	name := strings.TrimSpace(r.FormValue("difficulty"))
	if name == "" {
		log.Printf("From AddDifficultyHandler : field can't be empty")
		errorMessage := url.QueryEscape("Le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	err := queries.CreateDifficulty(r.Context(), db.CreateDifficultyParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From AddDifficultyHandler : CreateDifficulty DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.DifficultiesURL, http.StatusSeeOther)
}

func EditFormDifficultyHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	difficultyIDStr := r.FormValue("difficulty_id")
	if difficultyIDStr == "" {
		http.Error(w, "From EditFormDifficultyHandler : No difficulty id parameter", http.StatusBadRequest)
		return
	}

	difficultyID, err := strconv.ParseInt(difficultyIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditFormDifficultyHandler : Invalid difficulty ID", http.StatusBadRequest)
		return
	}

	difficulty, err := queries.GetDifficultyNameByID(r.Context(), db.GetDifficultyNameByIDParams{
		ID:     difficultyID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditFormDifficultyHandler : GetDifficultyNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.DifficultyPageData{
		Routes:           data.DefaultDashboardRoutes,
		DifficultyRoutes: data.DefaultDifficultyRoutes,
		PageTitle:        "edit difficulty",
		ExtraData: map[string]any{
			"Difficulty":   difficulty,
			"DifficultyID": difficultyIDStr,
		},
	}
	RenderEditFormDifficulty(w, dataPage)
}

func EditDifficultyHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	newDifficulty := strings.TrimSpace(r.FormValue("new_difficulty"))
	if newDifficulty == "" {
		log.Printf("From EditDifficultyHandler : field can't be empty")
		errorMessage := url.QueryEscape("Le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	difficultyIDStr := strings.TrimSpace(r.FormValue("difficulty_id"))
	if difficultyIDStr == "" {
		http.Error(w, "From EditDifficultyHandler : DifficultyID missing", http.StatusInternalServerError)
		return
	}
	difficultyID, err := strconv.ParseInt(difficultyIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditDifficultyHandler : Invalid skill ID", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateDifficulty(r.Context(), db.UpdateDifficultyParams{
		Name:   newDifficulty,
		ID:     difficultyID,
		UserID: userID,
	}); err != nil {
		log.Printf("From EditDifficultyHandler : UpdateDifficulty DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.DifficultiesURL, http.StatusSeeOther)
}

func DeleteFormDifficultyHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	difficultyIDStr := r.FormValue("difficulty_id")
	if difficultyIDStr == "" {
		http.Error(w, "From DeleteFormDifficultyHandler : No difficulty id parameter", http.StatusBadRequest)
		return
	}

	difficultyID, err := strconv.ParseInt(difficultyIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteFormDifficultyHandler : Invalid skill ID", http.StatusBadRequest)
		return
	}

	difficulty, err := queries.GetDifficultyNameByID(r.Context(), db.GetDifficultyNameByIDParams{
		ID:     difficultyID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormDifficultyHandler : GetDifficultyNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.DifficultyPageData{
		Routes:           data.DefaultDashboardRoutes,
		DifficultyRoutes: data.DefaultDifficultyRoutes,
		PageTitle:        "delete difficulty",
		ExtraData: map[string]any{
			"Difficulty":   difficulty,
			"DifficultyID": difficultyIDStr,
		},
	}

	RenderDeleteFormDifficulty(w, dataPage)
}

func DeleteDifficultyHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	difficultyIDStr := r.FormValue("difficulty_id")
	if difficultyIDStr == "" {
		http.Error(w, "From DeleteDifficultyHandler : No difficulty id parameter", http.StatusBadRequest)
		return
	}

	difficultyID, err := strconv.ParseInt(difficultyIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteDifficultyHandler : Invalid difficulty ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteDifficulty(r.Context(), db.DeleteDifficultyParams{
		ID:     difficultyID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteDifficultyHandler : DeleteDifficulty DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.DifficultiesURL, http.StatusSeeOther)
}
