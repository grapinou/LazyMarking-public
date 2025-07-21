package skills

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

func TableSkillsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	skillsDB, err := queries.GetAllSkills(r.Context(), userID)
	if err != nil {
		log.Printf("From TableSkillsHandler, GetAllSkills DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	noSkill := true
	if len(skillsDB) > 0 {
		noSkill = false
	}

	var actionsURLParameters []data.SkillActionURLs
	if !noSkill {
		for _, skill := range skillsDB {
			editURL := data.DefaultSkillRoutes.EditURL + "?skill_id=" + url.QueryEscape(strconv.FormatInt(skill.ID, 10))
			deleteURL := data.DefaultSkillRoutes.DeleteURL + "?skill_id=" + url.QueryEscape(strconv.FormatInt(skill.ID, 10))

			actionsURLParameters = append(actionsURLParameters, data.SkillActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.SkillPageData{
		Routes:      data.DefaultDashboardRoutes,
		SkillRoutes: data.DefaultSkillRoutes,
		PageTitle:   "skills",
		ExtraData: map[string]any{
			"NoSkill": noSkill,
			"Action":  actionsURLParameters,
			"Skills":  skillsDB,
		},
	}

	RenderTableSkillPage(w, dataPage)
}

func AddFormSkillHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	dataPage := data.SkillPageData{
		Routes:      data.DefaultDashboardRoutes,
		SkillRoutes: data.DefaultSkillRoutes,
		PageTitle:   "add skill",
	}
	RenderAddFormSkill(w, dataPage)
}

func AddSkillHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	name := strings.TrimSpace(r.FormValue("skill"))
	if name == "" {
		log.Printf("From AddSkillHandler : name field can't be empty")
		errorMessage := url.QueryEscape("Le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	err := queries.CreateSkill(r.Context(), db.CreateSkillParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From AddSkillHandler, CreateSkill : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.SkillsURL, http.StatusSeeOther)
}

func EditFormSkillHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	skillIDStr := r.FormValue("skill_id")
	if skillIDStr == "" {
		http.Error(w, "From EditFormSkillHandler : no skill id parameter", http.StatusBadRequest)
		return
	}

	skillID, err := strconv.ParseInt(skillIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditFormSkillHandler : invalid skill ID", http.StatusBadRequest)
		return
	}

	skill, err := queries.GetSkillNameByID(r.Context(), db.GetSkillNameByIDParams{
		ID:     skillID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditFormSkillHandler : GetSkillNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.SkillPageData{
		Routes:      data.DefaultDashboardRoutes,
		SkillRoutes: data.DefaultSkillRoutes,
		PageTitle:   "edit skill",
		ExtraData: map[string]any{
			"Skill":   skill,
			"SkillID": skillIDStr,
		},
	}
	RenderEditFormSkill(w, dataPage)
}

func EditSkillHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	newSkill := strings.TrimSpace(r.FormValue("new_skill"))
	if newSkill == "" {
		log.Printf("From EditSkillHandler : field can't be empty")
		errorMessage := url.QueryEscape("Le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	skillIDStr := strings.TrimSpace(r.FormValue("skill_id"))
	if skillIDStr == "" {
		http.Error(w, "From EditSkillHandler : skillID missing", http.StatusInternalServerError)
		return
	}
	skillID, err := strconv.ParseInt(skillIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditSkillHandler : invalid skill ID", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateSkill(r.Context(), db.UpdateSkillParams{
		Name:   newSkill,
		ID:     skillID,
		UserID: userID,
	}); err != nil {
		log.Printf("From EditSkillHandler : UpdateSkill DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.SkillsURL, http.StatusSeeOther)
}

func DeleteFormSkillHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	skillIDStr := r.FormValue("skill_id")
	if skillIDStr == "" {
		http.Error(w, "From DeleteFormSkillHandler : no skill id parameter", http.StatusBadRequest)
		return
	}

	skillID, err := strconv.ParseInt(skillIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteFormSkillHandler : invalid skill ID", http.StatusBadRequest)
		return
	}

	skill, err := queries.GetSkillNameByID(r.Context(), db.GetSkillNameByIDParams{
		ID:     skillID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormSkillHandler : GetSkillNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.SkillPageData{
		Routes:      data.DefaultDashboardRoutes,
		SkillRoutes: data.DefaultSkillRoutes,
		PageTitle:   "delete skill",
		ExtraData: map[string]any{
			"Skill":   skill,
			"SkillID": skillIDStr,
		},
	}

	RenderDeleteFormSkill(w, dataPage)
}

func DeleteSkillHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	skillIDStr := r.FormValue("skill_id")
	if skillIDStr == "" {
		http.Error(w, "From DeleteSkillHandler : no skill id parameter", http.StatusBadRequest)
		return
	}

	skillID, err := strconv.ParseInt(skillIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteSkillHandler : invalid skill ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteSkill(r.Context(), db.DeleteSkillParams{
		ID:     skillID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteSkillHandler : DeleteSkill DB error: %v", err)
		http.Error(w, "From DeleteSkillHandler : Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.SkillsURL, http.StatusSeeOther)
}
