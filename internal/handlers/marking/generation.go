package marking

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"mime"
	"net/http"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func markingGenerationURL(id int64) string {
	return data.DefaultMarkingRoutes.GenerationResults + "?exam_generated_id=" + strconv.FormatInt(id, 10)
}

func loadMarkingGeneration(ctx context.Context, queries *db.Queries, userID, generationID int64) (data.MarkingGenerationPageData, error) {
	generation, err := queries.GetMarkingGeneration(ctx, db.GetMarkingGenerationParams{UserID: userID, GenerationID: generationID})
	if err != nil {
		return data.MarkingGenerationPageData{}, err
	}
	rows, err := queries.ListCurrentExamResultsForGeneration(ctx, db.ListCurrentExamResultsForGenerationParams{UserID: userID, GenerationID: generationID})
	if err != nil {
		return data.MarkingGenerationPageData{}, err
	}
	summary := buildMarkingExamSummary(rows)
	param := "?exam_generated_id=" + strconv.FormatInt(generationID, 10)
	return data.MarkingGenerationPageData{
		Routes: data.DefaultDashboardRoutes, PageTitle: tools.MarkingGenerationTitle(generation.ClassName, generation.ExamName),
		GenerationID: generationID, ExamName: generation.ExamName, ClassName: generation.ClassName,
		AddCopiesURL: data.DefaultDashboardRoutes.MarkingURL + param,
		PDFURL:       data.DefaultMarkingRoutes.GenerationPDF + param,
		Summary:      summary,
		Pedagogy:     buildMarkingPedagogicalSummary(rows),
		Progress:     buildMarkingProgress(summary),
	}, nil
}

func MarkingGenerationHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	serveMarkingGeneration(w, r, queries, false)
}

func MarkingGenerationPDFHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	serveMarkingGeneration(w, r, queries, true)
}

func serveMarkingGeneration(w http.ResponseWriter, r *http.Request, queries *db.Queries, pdf bool) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}
	id, ok := parsePositiveReviewFormInt(r.URL.Query().Get("exam_generated_id"))
	if !ok {
		http.Error(w, "Évaluation invalide.", http.StatusBadRequest)
		return
	}
	page, err := loadMarkingGeneration(r.Context(), queries, userID, id)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("load marking generation: %v", err)
		http.Error(w, "Impossible de charger le bilan.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if pdf {
		content, err := tools.BuildMarkingGenerationPDF(r.Context(), username, page)
		if err != nil {
			log.Printf("generate cumulative marking PDF: %v", err)
			http.Error(w, "Impossible de produire le bilan PDF. Veuillez réessayer.", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
			"filename": tools.MarkingGenerationReportFilename(page.ClassName, page.ExamName),
		}))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write(content)
		return
	}
	imports, err := queries.ListMarkingJobHistory(r.Context(), db.ListMarkingJobHistoryParams{UserID: userID, GenerationID: id})
	if err != nil {
		log.Printf("load generation import history: %v", err)
		http.Error(w, "Impossible de charger les imports.", http.StatusInternalServerError)
		return
	}
	page.Imports = buildMarkingJobHistory(imports)
	tools.RenderMergeTemplate(w, page, data.DefaultDashboarPath, data.DefaultDashboardName, data.DefaultMarkingPathTemplate, "generation_results.html")
}
