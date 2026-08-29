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

var (
	renderTableSkillPage      = RenderTableSkillPage
	renderAddFormSkillPage    = RenderAddFormSkillPage
	renderEditFormSkillPage   = RenderEditFormSkillPage
	renderDeleteFormSkillPage = RenderDeleteFormSkillPage
)

func TableSkillsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableSkillsHandler -> tools.CheckRequest return not ok")
		return
	}

	skillsDB, err := queries.GetAllSkills(r.Context(), userID)
	if err != nil {
		log.Printf("From TableSkillsHandler -> GetAllSkills DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	items := make([]data.SkillListItem, 0, len(skillsDB))
	for _, skill := range skillsDB {
		items = append(items, data.SkillListItem{
			ID: skill.ID, Name: skill.Name,
			EditURL: data.SkillURL(data.DefaultSkillRoutes.EditURL, skill.ID), DeleteURL: data.SkillURL(data.DefaultSkillRoutes.DeleteURL, skill.ID),
		})
	}

	dataPage := data.SkillPageData{
		Routes:      data.DefaultDashboardRoutes,
		SkillRoutes: data.DefaultSkillRoutes,
		PageTitle:   "Compétences", SkillItems: items,
	}
	renderTableSkillPage(w, dataPage)
}

func AddFormSkillHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormSkillHandler -> tools.CheckRequest return not ok")
		return
	}

	dataPage := data.SkillPageData{
		Routes:      data.DefaultDashboardRoutes,
		SkillRoutes: data.DefaultSkillRoutes,
		CancelURL:   data.DefaultQuestionRoutes.SkillsURL, PageTitle: "Ajouter une compétence",
	}
	renderAddFormSkillPage(w, dataPage)
}

func AddSkillHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddSkillHandler -> tools.CheckRequest return not ok")
		return
	}

	name := strings.TrimSpace(r.FormValue("skill"))

	err := queries.CreateSkill(r.Context(), db.CreateSkillParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From AddSkillHandler -> CreateSkill : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ ou le champ ne peut être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.SkillsURL, http.StatusSeeOther)
}

func EditFormSkillHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormSkillHandler -> tools.CheckRequest return not ok")
		return
	}

	skillIDStr := r.URL.Query().Get("skill_id")
	if skillIDStr == "" {
		log.Println("From EditFormSkillHandler : no skill id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	skillID, err := strconv.ParseInt(skillIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormSkillHandler -> strconv.ParseInt, invalid skill ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	skill, err := queries.GetSkillNameByID(r.Context(), db.GetSkillNameByIDParams{
		ID:     skillID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormSkillHandler GetSkillNameByID")
		return
	}

	dataPage := data.SkillPageData{
		Routes:       data.DefaultDashboardRoutes,
		SkillRoutes:  data.DefaultSkillRoutes,
		SkillContext: data.SkillContext{ID: skillID, Name: skill}, CancelURL: data.DefaultQuestionRoutes.SkillsURL, PageTitle: "Modifier la compétence",
	}
	renderEditFormSkillPage(w, dataPage)
}

func EditSkillHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditSkillHandler -> tools.CheckRequest return not ok")
		return
	}

	newSkill := strings.TrimSpace(r.FormValue("new_skill"))

	skillIDStr := r.FormValue("skill_id")
	if skillIDStr == "" {
		log.Println("From EditSkillHandler : no skillID")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	skillID, err := strconv.ParseInt(skillIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditSkillHandler -> strconv.ParseInt, invalid skillID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.UpdateSkill(r.Context(), db.UpdateSkillParams{
		Name:   newSkill,
		ID:     skillID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditSkillHandler : UpdateSkill DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ ou le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "UpdateSkill") {
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.SkillsURL, http.StatusSeeOther)
}

func DeleteFormSkillHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableSkillsHandler -> tools.CheckRequest return not ok")
		return
	}

	skillIDStr := r.URL.Query().Get("skill_id")
	if skillIDStr == "" {
		log.Println("From DeleteFormSkillHandler : no skill id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	skillID, err := strconv.ParseInt(skillIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormSkillHandler -> strconv.ParseInt, invalid skill ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	skill, err := queries.GetSkillNameByID(r.Context(), db.GetSkillNameByIDParams{
		ID:     skillID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteFormSkillHandler GetSkillNameByID")
		return
	}

	dataPage := data.SkillPageData{
		Routes:       data.DefaultDashboardRoutes,
		SkillRoutes:  data.DefaultSkillRoutes,
		SkillContext: data.SkillContext{ID: skillID, Name: skill}, CancelURL: data.DefaultQuestionRoutes.SkillsURL, PageTitle: "Supprimer la compétence",
	}
	renderDeleteFormSkillPage(w, dataPage)
}

func DeleteSkillHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteSkillHandler -> tools.CheckRequest return not ok")
		return
	}

	skillIDStr := r.FormValue("skill_id")
	if skillIDStr == "" {
		log.Println("From DeleteSkillHandler : no skill id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	skillID, err := strconv.ParseInt(skillIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteSkillHandler -> strconv.ParseInt, invalid skill ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.DeleteSkill(r.Context(), db.DeleteSkillParams{
		ID:     skillID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteSkillHandler : DeleteSkill DB error: %v", err)
		if tools.IsSQLiteForeignKeyConstraint(err) {
			errorMessage := url.QueryEscape("Ce champ est utilisé par une question. Impossible de le supprimer pour l'instant.")
			http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
			return
		}
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "DeleteSkill") {
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.SkillsURL, http.StatusSeeOther)
}
