package skills

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func SkillsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	skillsDB, err := queries.GetAllSkills(r.Context(), userID)
	if err != nil {
		log.Printf("GetAllSkills DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	log.Println("skillsDB", skillsDB)
	noSubject := true
	if len(skillsDB) > 0 {
		noSubject = false
	}

	var actionsURLParameters []data.SkillActionURLs
	if !noSubject {
		for _, skill := range skillsDB {
			editURL := data.DefaultSkillRoutes.EditURL + "?skill_id=" + url.QueryEscape(strconv.FormatInt(skill.ID, 10))
			deleteURL := data.DefaultSkillRoutes.DeleteURL + "?skill_id=" + url.QueryEscape(strconv.FormatInt(skill.ID, 10))

			actionsURLParameters = append(actionsURLParameters, data.SkillActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	data := data.SkillPageData{
		Routes:      data.DefaultDashboardRoutes,
		SkillRoutes: data.DefaultSkillRoutes,
		PageTitle:   "skills",
		ExtraData: map[string]any{
			"NoSubject": noSubject,
			"Action":    actionsURLParameters,
			"Skills":    skillsDB,
		},
	}

	RenderSkillPage(w, data)
}

func AddSkillsFormHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	data := data.SkillPageData{
		SkillRoutes: data.DefaultSkillRoutes,
		PageTitle:   "add skill",
	}
	RenderAddSkillForm(w, data)
}

func AddSkillsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	name := strings.TrimSpace(r.FormValue("skill"))
	if name == "" {
		http.Error(w, "Name field can't be empty", http.StatusBadRequest)
		return
	}

	err := queries.CreateSkill(r.Context(), db.CreateSkillParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("CreateSkill DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.SkillsURL, http.StatusSeeOther)
}

func EditSkillsFormHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	skillIDStr := r.FormValue("skill_id")
	if skillIDStr == "" {
		http.Error(w, "No skill id parameter", http.StatusBadRequest)
		return
	}

	skillID, err := strconv.ParseInt(skillIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid skill ID", http.StatusBadRequest)
		return
	}

	skill, err := queries.GetSkillNameByID(r.Context(), db.GetSkillNameByIDParams{
		ID:     skillID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("GetSkillNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	data := data.SkillPageData{
		SkillRoutes: data.DefaultSkillRoutes,
		PageTitle:   "edit skill",
		ExtraData: map[string]any{
			"Skill":   skill,
			"SkillID": skillIDStr,
		},
	}
	RenderEditSkillForm(w, data)
}

func EditSkillsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	newSkill := strings.TrimSpace(r.FormValue("new_skill"))
	if newSkill == "" {
		http.Error(w, "Skill field can't be empty", http.StatusBadRequest)
		return
	}

	skillIDStr := strings.TrimSpace(r.FormValue("skill_id"))
	if skillIDStr == "" {
		http.Error(w, "SkillID missing", http.StatusInternalServerError)
		return
	}
	skillID, err := strconv.ParseInt(skillIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid skill ID", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateSkill(r.Context(), db.UpdateSkillParams{
		Name:   newSkill,
		ID:     skillID,
		UserID: userID,
	}); err != nil {
		log.Printf("UpdateSkill DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.SkillsURL, http.StatusSeeOther)
}

func DeleteFormSkillsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	skillIDStr := r.FormValue("skill_id")
	if skillIDStr == "" {
		http.Error(w, "No skill id parameter", http.StatusBadRequest)
		return
	}

	skillID, err := strconv.ParseInt(skillIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid skill ID", http.StatusBadRequest)
		return
	}

	skill, err := queries.GetSkillNameByID(r.Context(), db.GetSkillNameByIDParams{
		ID:     skillID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("GetSkillNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	data := data.SkillPageData{
		SkillRoutes: data.DefaultSkillRoutes,
		PageTitle:   "delete skill",
		ExtraData: map[string]any{
			"Skill":   skill,
			"SkillID": skillIDStr,
		},
	}

	RenderDeleteSkillForm(w, data)
}

func DeleteSkillsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	skillIDStr := r.FormValue("skill_id")
	if skillIDStr == "" {
		http.Error(w, "No skill id parameter", http.StatusBadRequest)
		return
	}

	skillID, err := strconv.ParseInt(skillIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid skill ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteSkill(r.Context(), db.DeleteSkillParams{
		ID:     skillID,
		UserID: userID,
	}); err != nil {
		log.Printf("DeleteSkill DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.SkillsURL, http.StatusSeeOther)
}
