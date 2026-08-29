package themes

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
	renderTableThemePage      = RenderTableThemePage
	renderAddThemeFormPage    = RenderAddThemeFormPage
	renderEditFormThemePage   = RenderEditFormThemePage
	renderDeleteFormThemePage = RenderDeleteFormThemePage
)

func TableThemesHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableThemesHandler -> tools.CheckRequest return not ok")
		return
	}

	themesDB, err := queries.GetAllThemes(r.Context(), userID)
	if err != nil {
		log.Printf("From TableThemesHandler -> GetAllTheme DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	items := make([]data.ThemeListItem, 0, len(themesDB))
	for _, theme := range themesDB {
		items = append(items, data.ThemeListItem{
			ID: theme.ID, Name: theme.Name,
			EditURL: data.ThemeURL(data.DefaultThemeRoutes.EditURL, theme.ID), DeleteURL: data.ThemeURL(data.DefaultThemeRoutes.DeleteURL, theme.ID),
		})
	}

	dataPage := data.ThemePageData{
		Routes:      data.DefaultDashboardRoutes,
		ThemeRoutes: data.DefaultThemeRoutes,
		PageTitle:   "Thèmes", ThemeItems: items,
	}
	renderTableThemePage(w, dataPage)
}

func AddFormThemeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormThemeHandler -> tools.CheckRequest return not ok")
		return
	}

	dataPage := data.ThemePageData{
		Routes:      data.DefaultDashboardRoutes,
		ThemeRoutes: data.DefaultThemeRoutes,
		CancelURL:   data.DefaultQuestionRoutes.ThemesURL, PageTitle: "Ajouter un thème",
	}
	renderAddThemeFormPage(w, dataPage)
}

func AddThemeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddThemeHandler -> tools.CheckRequest return not ok")
		return
	}

	name := strings.TrimSpace(r.FormValue("theme"))

	err := queries.CreateTheme(r.Context(), db.CreateThemeParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From AddThemeHandler -> CreateTheme : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ ou le champ ne peut être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.ThemesURL, http.StatusSeeOther)
}

func EditFormThemeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormThemeHandler -> tools.CheckRequest return not ok")
		return
	}

	themeIDStr := r.URL.Query().Get("theme_id")
	if themeIDStr == "" {
		log.Printf("From EditFormThemeHandler : no theme id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	themeID, err := strconv.ParseInt(themeIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormThemeHandler -> strconv.ParseInt, invalid theme ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	theme, err := queries.GetThemeNameByID(r.Context(), db.GetThemeNameByIDParams{
		ID:     themeID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormThemeHandler GetThemeNameByID")
		return
	}

	dataPage := data.ThemePageData{
		Routes:       data.DefaultDashboardRoutes,
		ThemeRoutes:  data.DefaultThemeRoutes,
		ThemeContext: data.ThemeContext{ID: themeID, Name: theme}, CancelURL: data.DefaultQuestionRoutes.ThemesURL, PageTitle: "Modifier le thème",
	}
	renderEditFormThemePage(w, dataPage)
}

func EditThemeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditThemeHandler -> tools.CheckRequest return not ok")
		return
	}

	newTheme := strings.TrimSpace(r.FormValue("new_theme"))

	themeIDStr := r.FormValue("theme_id")
	if themeIDStr == "" {
		log.Printf("From EditThemeHandler : no theme id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	themeID, err := strconv.ParseInt(themeIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditThemeHandler -> strconv.ParseInt, invalid theme ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.UpdateTheme(r.Context(), db.UpdateThemeParams{
		Name:   newTheme,
		ID:     themeID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditThemeHandler -> UpdateTheme DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ ou le champ ne peut être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "UpdateTheme") {
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.ThemesURL, http.StatusSeeOther)
}

func DeleteFormThemeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormThemeHandler -> tools.CheckRequest return not ok")
		return
	}

	themeIDStr := r.URL.Query().Get("theme_id")
	if themeIDStr == "" {
		log.Println("From DeleteFormThemeHandler : No theme id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	themeID, err := strconv.ParseInt(themeIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormThemeHandler : Invalid theme ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	theme, err := queries.GetThemeNameByID(r.Context(), db.GetThemeNameByIDParams{
		ID:     themeID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteFormThemeHandler GetThemeNameByID")
		return
	}

	dataPage := data.ThemePageData{
		Routes:       data.DefaultDashboardRoutes,
		ThemeRoutes:  data.DefaultThemeRoutes,
		ThemeContext: data.ThemeContext{ID: themeID, Name: theme}, CancelURL: data.DefaultQuestionRoutes.ThemesURL, PageTitle: "Supprimer le thème",
	}
	renderDeleteFormThemePage(w, dataPage)
}

func DeleteThemeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteThemeHandler -> tools.CheckRequest return not ok")
		return
	}

	themeIDStr := r.FormValue("theme_id")
	if themeIDStr == "" {
		log.Println("From DeleteThemeHandler : No theme id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	themeID, err := strconv.ParseInt(themeIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteThemeHandler -> strconv.ParseInt, Invalid theme ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.DeleteTheme(r.Context(), db.DeleteThemeParams{
		ID:     themeID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteThemeHandler : DeleteTheme DB error: %v", err)
		if tools.IsSQLiteForeignKeyConstraint(err) {
			errorMessage := url.QueryEscape("Ce champ est utilisé par une question. Impossible de le supprimer pour l'instant.")
			http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
			return
		}
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "DeleteTheme") {
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.ThemesURL, http.StatusSeeOther)
}
