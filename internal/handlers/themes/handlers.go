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

func TableThemesHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	themesDB, err := queries.GetAllThemes(r.Context(), userID)
	if err != nil {
		log.Printf("GetAllTheme DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	noSubject := true
	if len(themesDB) > 0 {
		noSubject = false
	}

	var actionsURLParameters []data.ThemeActionURLs
	if !noSubject {
		for _, theme := range themesDB {
			editURL := data.DefaultThemeRoutes.EditURL + "?theme_id=" + url.QueryEscape(strconv.FormatInt(theme.ID, 10))
			deleteURL := data.DefaultThemeRoutes.DeleteURL + "?theme_id=" + url.QueryEscape(strconv.FormatInt(theme.ID, 10))

			actionsURLParameters = append(actionsURLParameters, data.ThemeActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.ThemePageData{
		Routes:      data.DefaultDashboardRoutes,
		ThemeRoutes: data.DefaultThemeRoutes,
		PageTitle:   "themes",
		ExtraData: map[string]any{
			"NoSubject": noSubject,
			"Action":    actionsURLParameters,
			"Themes":    themesDB,
		},
	}

	RenderTableThemePage(w, dataPage)
}

func AddFormThemeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	dataPage := data.ThemePageData{
		ThemeRoutes: data.DefaultThemeRoutes,
		PageTitle:   "add theme",
	}
	RenderAddThemeForm(w, dataPage)
}

func AddThemeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	name := strings.TrimSpace(r.FormValue("theme"))
	if name == "" {
		http.Error(w, "Name field can't be empty", http.StatusBadRequest)
		return
	}

	err := queries.CreateTheme(r.Context(), db.CreateThemeParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("CreateTheme DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.ThemesURL, http.StatusSeeOther)
}

func EditFormThemeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	themeIDStr := r.FormValue("theme_id")
	if themeIDStr == "" {
		http.Error(w, "No theme id parameter", http.StatusBadRequest)
		return
	}

	themeID, err := strconv.ParseInt(themeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid theme ID", http.StatusBadRequest)
		return
	}

	theme, err := queries.GetThemeNameByID(r.Context(), db.GetThemeNameByIDParams{
		ID:     themeID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("GetThemeNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.ThemePageData{
		ThemeRoutes: data.DefaultThemeRoutes,
		PageTitle:   "edit theme",
		ExtraData: map[string]any{
			"Theme":   theme,
			"ThemeID": themeIDStr,
		},
	}
	RenderEditFormTheme(w, dataPage)
}

func EditThemeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	newTheme := strings.TrimSpace(r.FormValue("new_theme"))
	if newTheme == "" {
		http.Error(w, "Theme field can't be empty", http.StatusBadRequest)
		return
	}

	themeIDStr := strings.TrimSpace(r.FormValue("theme_id"))
	if themeIDStr == "" {
		http.Error(w, "ThemeID missing", http.StatusInternalServerError)
		return
	}
	themeID, err := strconv.ParseInt(themeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid theme ID", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateTheme(r.Context(), db.UpdateThemeParams{
		Name:   newTheme,
		ID:     themeID,
		UserID: userID,
	}); err != nil {
		log.Printf("UpdateTheme DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.ThemesURL, http.StatusSeeOther)
}

func DeleteFormThemeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	themeIDStr := r.FormValue("theme_id")
	if themeIDStr == "" {
		http.Error(w, "No theme id parameter", http.StatusBadRequest)
		return
	}

	themeID, err := strconv.ParseInt(themeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid theme ID", http.StatusBadRequest)
		return
	}

	theme, err := queries.GetThemeNameByID(r.Context(), db.GetThemeNameByIDParams{
		ID:     themeID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("GetThemeNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.ThemePageData{
		ThemeRoutes: data.DefaultThemeRoutes,
		PageTitle:   "delete theme",
		ExtraData: map[string]any{
			"Theme":   theme,
			"ThemeID": themeIDStr,
		},
	}

	RenderDeleteFormTheme(w, dataPage)
}

func DeleteThemeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	themeIDStr := r.FormValue("theme_id")
	if themeIDStr == "" {
		http.Error(w, "No theme id parameter", http.StatusBadRequest)
		return
	}

	themeID, err := strconv.ParseInt(themeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid theme ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteTheme(r.Context(), db.DeleteThemeParams{
		ID:     themeID,
		UserID: userID,
	}); err != nil {
		log.Printf("DeleteTheme DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.ThemesURL, http.StatusSeeOther)
}
