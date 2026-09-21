package questions

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/questionfamilies"
	"github.com/grapinou/LazyMarking/internal/sharedlibrary"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterSharingRoutes(mux *http.ServeMux, conn *sql.DB) {
	for _, route := range []struct {
		pattern string
		handler func(http.ResponseWriter, *http.Request, *sql.DB)
	}{
		{"GET /dashboard/library", SharedLibraryHandler},
		{"GET /dashboard/library/preview", SharedPreviewHandler},
		{"GET /dashboard/library/image", SharedImageHandler},
		{"POST /dashboard/library/copy", CopySharedHandler},
		{"POST /dashboard/questions/sharing", QuestionSharingHandler},
	} {
		mux.Handle(route.pattern, login.CheckAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { route.handler(w, r, conn) })))
	}
}

func sharingError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	log.Printf("Question library: %v", err)
	http.Error(w, "Impossible de traiter cette ressource. Veuillez réessayer.", http.StatusInternalServerError)
}

func sharingID(w http.ResponseWriter, r *http.Request, post bool) (int64, bool) {
	value := r.URL.Query().Get("question_id")
	if post {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Formulaire invalide.", 400)
			return 0, false
		}
		value = r.PostForm.Get("question_id")
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Question invalide.", 400)
		return 0, false
	}
	return id, true
}

func QuestionSharingHandler(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}
	id, ok := sharingID(w, r, true)
	if !ok {
		return
	}
	state := r.PostForm.Get("shared")
	if state != "0" && state != "1" {
		http.Error(w, "Visibilité invalide.", 400)
		return
	}
	tx, err := conn.BeginTx(r.Context(), nil)
	if err != nil {
		sharingError(w, r, err)
		return
	}
	defer tx.Rollback()
	q := db.New(tx)
	if _, err := q.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{ID: id, UserID: userID}); err != nil {
		sharingError(w, r, err)
		return
	}
	if state == "1" {
		_, err = q.ShareQuestion(r.Context(), db.ShareQuestionParams{QuestionID: id, UserID: userID})
	} else {
		_, err = q.UnshareQuestion(r.Context(), db.UnshareQuestionParams{QuestionID: id, UserID: userID})
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		sharingError(w, r, err)
		return
	}
	http.Redirect(w, r, data.DefaultDashboardRoutes.QuestionsURL, http.StatusSeeOther)
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
	rows, err := q.GetSharedQuestions(r.Context())
	if err != nil {
		sharingError(w, r, err)
		return
	}
	alts, err := q.GetSharedVariants(r.Context())
	if err != nil {
		sharingError(w, r, err)
		return
	}
	var mains []questionfamilies.Question
	for _, m := range rows {
		mains = append(mains, questionfamilies.Question{ID: m.ID, Content: m.Content, Instruction: m.Instruction, Author: m.Author, Owned: m.UserID == userID, SubjectName: m.SubjectName, ThemeName: m.ThemeName, YearLevelName: m.YearLevelName, SkillName: m.SkillName, DifficultyName: m.DifficultyName, PointValue: m.PointValue})
	}
	var variants []questionfamilies.Variant
	for _, a := range alts {
		variants = append(variants, questionfamilies.Variant{ID: a.ID, QuestionID: a.QuestionID, Content: a.Content})
	}
	families := questionfamilies.Build(mains, variants)
	filters := libraryFilters(r.URL.Query(), families)
	page := data.QuestionPageData{Routes: data.DefaultDashboardRoutes, PageTitle: "Bibliothèque", ExtraData: map[string]any{"Library": filters, "FilterAction": data.DefaultDashboardRoutes.LibraryURL, "Total": len(families), "QuestionFamilies": filterLibrary(families, filters)}}
	tools.RenderMergeTemplate(w, page, data.DefaultDashboarPath, data.DefaultDashboardName, data.DefaultQuestionPathTemplate, "shared_library.html", data.DefaultQuestionPathTemplate+"library_filters.html")
}

func SharedPreviewHandler(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}
	id, ok := sharingID(w, r, false)
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
	family, err := sharedlibrary.LoadShared(r.Context(), db.New(tx), id)
	if err != nil {
		sharingError(w, r, err)
		return
	}
	page := data.QuestionPageData{Routes: data.DefaultDashboardRoutes, PageTitle: "Aperçu de la famille partagée", ExtraData: map[string]any{"Family": family, "OwnQuestion": family.Metadata.UserID == userID}}
	tools.RenderMergeTemplate(w, page, data.DefaultDashboarPath, data.DefaultDashboardName, data.DefaultQuestionPathTemplate, "shared_preview.html")
}

func SharedImageHandler(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	if _, _, ok := tools.CheckRequest(w, r, http.MethodGet); !ok {
		return
	}
	id, ok := sharingID(w, r, false)
	if !ok {
		return
	}
	variant, err := strconv.ParseInt(r.URL.Query().Get("variant_id"), 10, 64)
	if err != nil || variant < 0 {
		http.Error(w, "Image invalide.", 400)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	tx, err := conn.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		sharingError(w, r, err)
		return
	}
	defer tx.Rollback()
	family, err := sharedlibrary.LoadShared(r.Context(), db.New(tx), id)
	if err != nil {
		sharingError(w, r, err)
		return
	}
	for _, v := range family.Versions {
		if v.ID != variant || v.Image == nil {
			continue
		}
		file, err := sharedlibrary.OpenImage(config.ImageSavePath, v.Image.Name)
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
		http.ServeContent(w, r, v.Image.Name, info.ModTime(), file)
		return
	}
	http.NotFound(w, r)
}

func CopySharedHandler(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}
	id, ok := sharingID(w, r, true)
	if !ok {
		return
	}
	_, err := sharedlibrary.CopyShared(r.Context(), conn, config.ImageSavePath, id, userID)
	if errors.Is(err, sharedlibrary.ErrOwnFamily) {
		http.Error(w, "Cette question est déjà dans Mes questions. Vous pouvez la modifier depuis votre banque.", http.StatusConflict)
		return
	}
	if err != nil {
		sharingError(w, r, err)
		return
	}
	http.Redirect(w, r, data.DefaultDashboardRoutes.QuestionsURL+"?copied=1", http.StatusSeeOther)
}
