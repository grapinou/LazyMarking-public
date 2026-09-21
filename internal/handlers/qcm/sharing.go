package qcm

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/grapinou/LazyMarking/internal/classification"
	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/sharedlibrary"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterSharingRoutes(mux *http.ServeMux, conn *sql.DB) {
	for _, route := range []struct {
		pattern string
		handler func(http.ResponseWriter, *http.Request, *sql.DB)
	}{
		{"GET /dashboard/library/qcm", SharedLibraryHandler},
		{"GET /dashboard/library/qcm/preview", SharedPreviewHandler},
		{"GET /dashboard/library/qcm/image", SharedImageHandler},
		{"POST /dashboard/library/qcm/copy", CopySharedHandler},
		{"POST /dashboard/qcm/sharing", QCMSharingHandler},
	} {
		mux.Handle(route.pattern, login.CheckAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { route.handler(w, r, conn) })))
	}
}

func sharingError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	log.Printf("QCM library: %v", err)
	http.Error(w, "Impossible de traiter ce QCM. Veuillez réessayer.", http.StatusInternalServerError)
}

func sharingID(w http.ResponseWriter, r *http.Request, key string, post bool) (int64, bool) {
	value := r.URL.Query().Get(key)
	if post {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Formulaire invalide.", http.StatusBadRequest)
			return 0, false
		}
		value = r.PostForm.Get(key)
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Identifiant invalide.", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func QCMSharingHandler(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}
	id, ok := sharingID(w, r, "qcm_id", true)
	if !ok {
		return
	}
	state := r.PostForm.Get("shared")
	if state != "0" && state != "1" {
		http.Error(w, "Visibilité invalide.", http.StatusBadRequest)
		return
	}
	tx, err := conn.BeginTx(r.Context(), nil)
	if err != nil {
		sharingError(w, r, err)
		return
	}
	defer tx.Rollback()
	q := db.New(tx)
	if _, err := q.GetQCMNameByID(r.Context(), db.GetQCMNameByIDParams{ID: id, UserID: userID}); err != nil {
		sharingError(w, r, err)
		return
	}
	if state == "1" {
		_, err = q.ShareQCM(r.Context(), db.ShareQCMParams{QcmID: id, UserID: userID})
	} else {
		_, err = q.UnshareQCM(r.Context(), db.UnshareQCMParams{QcmID: id, UserID: userID})
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		sharingError(w, r, err)
		return
	}
	http.Redirect(w, r, data.DefaultDashboardRoutes.QcmURL, http.StatusSeeOther)
}

type sharedQCMItem struct {
	db.GetSharedQCMsRow
	Own              bool
	Subjects, Levels []string
}

type sharedQCMFilters struct {
	Search, Subject, Level string
	Subjects, Levels       []string
}

// appendLabel retains display spelling; search uses the common Unicode key.
func appendLabel(labels []string, value string) []string {
	if value != "" && !slices.Contains(labels, value) {
		labels = append(labels, value)
		slices.Sort(labels)
	}
	return labels
}

func SharedLibraryHandler(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	tx, err := conn.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		sharingError(w, r, err)
		return
	}
	defer tx.Rollback()
	q := db.New(tx)
	rows, err := q.GetSharedQCMs(r.Context())
	if err != nil {
		sharingError(w, r, err)
		return
	}
	features, err := q.GetSharedQCMClassifications(r.Context())
	if err != nil {
		sharingError(w, r, err)
		return
	}
	filter := sharedQCMFilters{Search: strings.TrimSpace(r.URL.Query().Get("q")), Subject: r.URL.Query().Get("subject"), Level: r.URL.Query().Get("level")}
	byQCM := make(map[int64]*sharedQCMItem)
	for _, row := range rows {
		byQCM[row.ID] = &sharedQCMItem{GetSharedQCMsRow: row, Own: row.UserID == userID}
	}
	for _, feature := range features {
		item := byQCM[feature.QcmID]
		if item == nil {
			continue
		}
		item.Subjects = appendLabel(item.Subjects, feature.SubjectName)
		item.Levels = appendLabel(item.Levels, feature.YearLevelName)
		filter.Subjects = appendLabel(filter.Subjects, feature.SubjectName)
		filter.Levels = appendLabel(filter.Levels, feature.YearLevelName)
	}
	var items []sharedQCMItem
	for _, row := range rows {
		item := byQCM[row.ID]
		if filter.Subject != "" && !slices.Contains(item.Subjects, filter.Subject) ||
			filter.Level != "" && !slices.Contains(item.Levels, filter.Level) {
			continue
		}
		text := item.Name + " " + item.Author + " " + strings.Join(item.Subjects, " ") + " " + strings.Join(item.Levels, " ")
		if strings.Contains(classification.Key(text), classification.Key(filter.Search)) {
			items = append(items, *item)
		}
	}
	page := data.DashboardPageData{Routes: data.DefaultDashboardRoutes, PageTitle: "Bibliothèque de QCM", ExtraData: map[string]any{"Items": items, "Total": len(rows), "Filter": filter}}
	tools.RenderMergeTemplate(w, page, data.DefaultDashboarPath, data.DefaultDashboardName, data.DefaultQCMPathTemplate, "shared_library.html")
}

func SharedPreviewHandler(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}
	id, ok := sharingID(w, r, "qcm_id", false)
	if !ok {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	tx, err := conn.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		sharingError(w, r, err)
		return
	}
	defer tx.Rollback()
	qcm, err := sharedlibrary.LoadSharedQCM(r.Context(), db.New(tx), id)
	if err != nil {
		sharingError(w, r, err)
		return
	}
	page := data.DashboardPageData{Routes: data.DefaultDashboardRoutes, PageTitle: "Aperçu du QCM partagé", ExtraData: map[string]any{"QCM": qcm, "OwnQCM": qcm.Metadata.UserID == userID}}
	tools.RenderMergeTemplate(w, page, data.DefaultDashboarPath, data.DefaultDashboardName, data.DefaultQCMPathTemplate, "shared_preview.html")
}

func SharedImageHandler(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	if _, _, ok := tools.CheckRequest(w, r, http.MethodGet); !ok {
		return
	}
	id, ok := sharingID(w, r, "qcm_id", false)
	if !ok {
		return
	}
	questionID, ok := sharingID(w, r, "question_id", false)
	if !ok {
		return
	}
	variant, err := strconv.ParseInt(r.URL.Query().Get("variant_id"), 10, 64)
	if err != nil || variant < 0 {
		http.Error(w, "Image invalide.", http.StatusBadRequest)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	tx, err := conn.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		sharingError(w, r, err)
		return
	}
	defer tx.Rollback()
	family, err := sharedlibrary.LoadSharedQCMFamily(r.Context(), db.New(tx), id, questionID)
	if err != nil {
		sharingError(w, r, err)
		return
	}
	for _, version := range family.Versions {
		if version.ID != variant || version.Image == nil {
			continue
		}
		file, err := sharedlibrary.OpenImage(config.ImageSavePath, version.Image.Name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			sharingError(w, r, err)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, version.Image.Name, info.ModTime(), file)
		return
	}
	http.NotFound(w, r)
}

func CopySharedHandler(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}
	id, ok := sharingID(w, r, "qcm_id", true)
	if !ok {
		return
	}
	_, err := sharedlibrary.CopySharedQCM(r.Context(), conn, config.ImageSavePath, id, userID)
	if errors.Is(err, sharedlibrary.ErrOwnQCM) {
		http.Error(w, "Ce QCM est déjà dans Mes QCM. Vous pouvez le modifier depuis votre espace personnel.", http.StatusConflict)
		return
	}
	if err != nil {
		sharingError(w, r, err)
		return
	}
	http.Redirect(w, r, data.DefaultDashboardRoutes.QcmURL+"?copied=1", http.StatusSeeOther)
}
