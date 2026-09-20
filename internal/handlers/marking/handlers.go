package marking

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func AddPdfFormMarkingHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddPdfFormMarkingHandler -> tools.CheckRequest return not ok")
		return
	}

	examsGeneratedSuccess, err := queries.GetExamsGeneratedSuccess(r.Context(), userID)
	if err != nil {
		log.Printf("From AddPdfFormMarkingHandler -> queries.GetExamsGeneratedSuccess error DB : %v", err)
		http.Error(w, "Une erreur est survenue. Veuillez réessayer.", http.StatusInternalServerError)
		return
	}
	generations, err := loadMarkingGenerationList(r.Context(), queries, userID, examsGeneratedSuccess)
	if err != nil {
		log.Printf("From AddPdfFormMarkingHandler -> loadMarkingGenerationList error DB : %v", err)
		http.Error(w, "Une erreur est survenue. Veuillez réessayer.", http.StatusInternalServerError)
		return
	}
	recentJobs, err := queries.ListRecentMarkingJobs(r.Context(), userID)
	if err != nil {
		log.Printf("From AddPdfFormMarkingHandler -> queries.ListRecentMarkingJobs error DB : %v", err)
		http.Error(w, "Une erreur est survenue. Veuillez réessayer.", http.StatusInternalServerError)
		return
	}

	noExamGenerated := true
	if len(generations) > 0 {
		noExamGenerated = false
	}

	dataPage := data.MarkingPageData{
		Routes:        data.DefaultDashboardRoutes,
		MarkingRoutes: data.DefaultMarkingRoutes,
		PageTitle:     "Correction des copies",
		ExtraData: map[string]any{
			"NoExamGenerated": noExamGenerated,
			"Exams":           generations,
		},
		RecentJobs: buildMarkingJobHistory(recentJobs),
	}

	if value := r.URL.Query().Get("exam_generated_id"); value != "" {
		id, valid := parsePositiveReviewFormInt(value)
		if !valid {
			http.Error(w, "Évaluation invalide.", http.StatusBadRequest)
			return
		}
		generation, err := queries.GetMarkingGeneration(r.Context(), db.GetMarkingGenerationParams{GenerationID: id, UserID: userID})
		if err != nil {
			tools.HandleOwnedLookupError(w, err, "marking upload generation")
			return
		}
		dataPage.SelectedGenerationID = id
		dataPage.SelectedGenerationName = generation.ExamName + " — " + generation.ClassName
	}

	RenderAddPdfFormMarkingPage(w, dataPage)
}

func ProcessingMarkingHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries, appCtx context.Context, markingJobs *sync.WaitGroup) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		http.Error(w, "Authentification requise.", http.StatusUnauthorized)
		return
	}

	if err := tools.PurgeExpiredMarkingJobs(r.Context(), queries, time.Now()); err != nil {
		log.Printf("From ProcessingMarkingHandler -> purge expired marking jobs: %v", err)
	}

	if err := parseMarkingMultipartForm(w, r, MaxMarkingUploadBytes); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			http.Error(w, "Le fichier est trop volumineux.", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Le formulaire envoyé est invalide.", http.StatusBadRequest)
		return
	}
	examGeneratedID, err := strconv.ParseInt(r.FormValue("exam_generated_id"), 10, 64)
	if err != nil || examGeneratedID <= 0 {
		http.Error(w, "L’évaluation générée sélectionnée est invalide.", http.StatusBadRequest)
		return
	}
	status, err := queries.GetExamStatus(r.Context(), db.GetExamStatusParams{
		ID:     examGeneratedID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Impossible d’accéder aux données demandées.", http.StatusInternalServerError)
		return
	}
	if status != "success" {
		http.Error(w, "Cette génération n’est pas encore prête pour la correction.", http.StatusConflict)
		return
	}

	file, err := tools.CheckPdfFile(r, MaxMarkingUploadBytes)
	if err != nil {
		http.Error(w, "Le fichier PDF est invalide.", http.StatusBadRequest)
		return
	}
	sourceFilename := ""
	if files := r.MultipartForm.File["pdffile"]; len(files) > 0 {
		sourceFilename = tools.MarkingSourceFilename(files[0].Filename)
	}

	stagedFile, err := os.CreateTemp("", "lazymarking-upload-*.pdf")
	if err != nil {
		http.Error(w, "Impossible de préparer le fichier envoyé.", http.StatusInternalServerError)
		return
	}
	if _, err = io.Copy(stagedFile, file); err != nil {
		file.Close()
		stagedFile.Close()
		os.Remove(stagedFile.Name())
		http.Error(w, "Impossible de préparer le fichier envoyé.", http.StatusInternalServerError)
		return
	}
	if err = file.Close(); err != nil {
		stagedFile.Close()
		os.Remove(stagedFile.Name())
		http.Error(w, "Impossible de terminer la réception du fichier.", http.StatusInternalServerError)
		return
	}
	if _, err = stagedFile.Seek(0, io.SeekStart); err != nil {
		stagedFile.Close()
		os.Remove(stagedFile.Name())
		http.Error(w, "Impossible de préparer le fichier envoyé.", http.StatusInternalServerError)
		return
	}

	releaseAdmission, admitted := globalMarkingJobAdmission.tryAcquire()
	if !admitted {
		stagedFile.Close()
		os.Remove(stagedFile.Name())
		http.Error(w, "Le serveur traite déjà plusieurs corrections. Veuillez réessayer dans quelques instants.", http.StatusServiceUnavailable)
		return
	}
	admissionTransferred := false
	defer func() {
		if !admissionTransferred {
			releaseAdmission()
		}
	}()

	if _, err = inspectMarkingPDF(r.Context(), stagedFile.Name()); err != nil {
		stagedFile.Close()
		os.Remove(stagedFile.Name())
		switch {
		case errors.Is(err, errTooManyMarkingPDFPages):
			http.Error(w, "Le fichier contient trop de pages pour être traité en une seule correction.", http.StatusUnprocessableEntity)
		case errors.Is(err, errMarkingPDFPageTooLarge):
			http.Error(w, "Une page du fichier est trop grande pour être traitée.", http.StatusUnprocessableEntity)
		default:
			http.Error(w, "Le fichier PDF est invalide.", http.StatusBadRequest)
		}
		return
	}

	jobDBID, err := queries.CreateHybridMarkingJob(r.Context(), db.CreateHybridMarkingJobParams{
		UserID: userID,
		ExamGeneratedID: sql.NullInt64{
			Int64: examGeneratedID,
			Valid: true,
		},
		ResultSchemaVersion:     sql.NullInt64{Int64: tools.MarkingResultSchemaVersion, Valid: true},
		MarkingAlgorithmVersion: sql.NullString{String: tools.MarkingAlgorithmVersion, Valid: true},
		DetectionThreshold:      sql.NullFloat64{Float64: tools.MarkingDetectionThreshold, Valid: true},
		AmbiguityDelta:          sql.NullFloat64{Float64: 0, Valid: true},
		ReviewPolicyVersion:     sql.NullString{String: tools.MarkingReviewPolicyVersion, Valid: true},
		V2RoiRadiusRatio:        sql.NullFloat64{Float64: tools.V2ROIRadiusRatio, Valid: true},
		V2DarkPixelThreshold:    sql.NullFloat64{Float64: tools.V2DarkPixelThreshold, Valid: true},
		V2DarkRatioThreshold:    sql.NullFloat64{Float64: tools.V2DarkRatioThreshold, Valid: true},
		V2ChromaPixelThreshold:  sql.NullFloat64{Float64: tools.V2ChromaPixelThreshold, Valid: true},
		V2ChromaRatioThreshold:  sql.NullFloat64{Float64: tools.V2ChromaRatioThreshold, Valid: true},
		SourcePdfFilename:       sql.NullString{String: sourceFilename, Valid: sourceFilename != ""},
	})
	if err != nil {
		stagedFile.Close()
		os.Remove(stagedFile.Name())
		log.Printf("From ProcessingMarkingHandler -> queries.CreateMarkingJob DB error : %v", err)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Cette génération n’est pas encore prête pour la correction.", http.StatusConflict)
			return
		}
		http.Error(w, "Une erreur est survenue. Veuillez réessayer.", http.StatusInternalServerError)
		return
	}

	// Lance la goroutine principale
	markingJobs.Add(1)
	admissionTransferred = true
	go func() {
		defer markingJobs.Done()
		defer releaseAdmission()
		defer stagedFile.Close()
		defer os.Remove(stagedFile.Name())
		tools.ProcessMarking(appCtx, userID, username, jobDBID, stagedFile, queries)
	}()

	params := "?job_id=" + url.QueryEscape(strconv.FormatInt(jobDBID, 10))
	propressMarkingURL := data.DefaultMarkingRoutes.ProgressMarking + params
	http.Redirect(w, r, propressMarkingURL, http.StatusSeeOther)
}

func ProgressMarkingHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		http.Error(w, "Authentification requise.", http.StatusUnauthorized)
		return
	}

	jobIDStr := r.URL.Query().Get("job_id")
	if jobIDStr == "" {
		log.Println("From ProgressMarkingHandler -> no job id")
		http.Error(w, "La requête est invalide ou incomplète.", http.StatusBadRequest)
		return
	}

	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		log.Printf("From ProgressMarkingHandler -> strconv.ParseInt invalid jobIDStr, error : %v", err)
		http.Error(w, "La requête est invalide ou incomplète.", http.StatusBadRequest)
		return
	}

	markingStatus, err := queries.GetMarkingStatus(r.Context(), db.GetMarkingStatusParams{
		ID:     jobID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		log.Printf("From ProgressMarkingHandler -> GetMarkingStatus : DB error : %v", err)
		http.Error(w, "Impossible d’accéder aux données demandées.", http.StatusInternalServerError)
		return
	}

	if markingStatus.Status == "failed" {
		log.Println("From ProgressMarkingHandler -> marking status failed")
		errorMessage := url.QueryEscape("La correction a échoué. Vérifiez que le PDF correspond à l’évaluation sélectionnée. Si le problème persiste, contactez l’administrateur.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	if markingStatus.Status == "success" && markingStatus.StatusPdf == "success" {
		params := "?job_id=" + url.QueryEscape(strconv.FormatInt(jobID, 10))
		successURL := data.DefaultMarkingRoutes.SuccessURL + params
		http.Redirect(w, r, successURL, http.StatusSeeOther)
		return
	}

	progress, err := queries.GetMarkingProgress(r.Context(), db.GetMarkingProgressParams{
		ID:     jobID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From ProgressMarkingHandler -> GetMarkingProgress : DB error : %v", err)
		http.Error(w, "Impossible d’accéder aux données demandées.", http.StatusInternalServerError)
		return
	}

	dataPage := data.MarkingPageData{
		Routes:        data.DefaultDashboardRoutes,
		MarkingRoutes: data.DefaultMarkingRoutes,
		PageTitle:     "Correction des copies",
		ExtraData: map[string]any{
			"JobID":         jobIDStr,
			"ProcessedPage": progress.DonePages.Int64,
			"TotalPages":    progress.TotalPages.Int64,
			"ProcessedExam": progress.DoneExams.Int64,
			"TotalExams":    progress.TotalExams.Int64,
			"Status":        markingStatus.Status,
			"StatusPdf":     markingStatus.StatusPdf,
		},
	}

	RenderProgressMarkingPage(w, dataPage)
}

func SuccessMarkingProcessingHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From SuccessMarkingProcessingHandler-> tools.CheckRequest return not ok")
		return
	}
	jobIDStr := r.URL.Query().Get("job_id")
	if jobIDStr == "" {
		log.Println("From SuccessMarkingProcessingHandler -> no job id")
		http.Error(w, "La requête est invalide ou incomplète.", http.StatusBadRequest)
		return
	}

	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		log.Printf("From SuccessMarkingProcessingHandler -> strconv.ParseInt invalid jobIDStr, error : %v", err)
		http.Error(w, "La requête est invalide ou incomplète.", http.StatusBadRequest)
		return
	}

	markingStatus, err := queries.GetMarkingStatus(r.Context(), db.GetMarkingStatusParams{
		ID:     jobID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		log.Printf("From SuccessMarkingProcessingHandler -> GetMarkingStatus DB error : %v", err)
		http.Error(w, "Une erreur est survenue. Veuillez réessayer.", http.StatusInternalServerError)
		return
	}
	if markingStatus.Status != "success" {
		http.NotFound(w, r)
		return
	}
	if markingStatus.StatusPdf != "success" {
		http.Redirect(w, r, data.DefaultMarkingRoutes.ProgressMarking+"?job_id="+url.QueryEscape(strconv.FormatInt(jobID, 10)), http.StatusSeeOther)
		return
	}

	generation, err := queries.GetMarkingJobGeneration(r.Context(), db.GetMarkingJobGenerationParams{MarkingJobID: jobID, UserID: userID})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "marking result generation")
		return
	}
	if generation.Valid && r.URL.Query().Get("import") != "1" && r.URL.Query().Get("notice") == "" {
		http.Redirect(w, r, markingGenerationURL(generation.Int64), http.StatusSeeOther)
		return
	}

	target, err := queries.GetMarkingArtifactsRegenerationTarget(r.Context(), db.GetMarkingArtifactsRegenerationTargetParams{MarkingJobID: jobID, UserID: userID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		log.Printf("From SuccessMarkingProcessingHandler -> GetMarkingArtifactsRegenerationTarget: %v", err)
		http.Error(w, "Une erreur est survenue. Veuillez réessayer.", http.StatusInternalServerError)
		return
	}
	summary, err := queries.GetMarkingReviewSummary(r.Context(), db.GetMarkingReviewSummaryParams{MarkingJobID: jobID, UserID: userID})
	if err != nil {
		log.Printf("From SuccessMarkingProcessingHandler -> GetMarkingReviewSummary: %v", err)
		http.Error(w, "Une erreur est survenue. Veuillez réessayer.", http.StatusInternalServerError)
		return
	}
	reviewStatus, err := db.DeriveMarkingReviewStatus(summary.AmbiguityDelta, summary.TotalCandidates, summary.PendingCandidates)
	if err != nil {
		log.Printf("From SuccessMarkingProcessingHandler -> DeriveMarkingReviewStatus: %v", err)
		http.Error(w, "Une erreur est survenue. Veuillez réessayer.", http.StatusInternalServerError)
		return
	}
	nonCorrected, err := queries.GetMarkingNonCorrectedSummary(r.Context(), db.GetMarkingNonCorrectedSummaryParams{MarkingJobID: jobID, UserID: userID})
	if err != nil {
		log.Printf("From SuccessMarkingProcessingHandler -> GetMarkingNonCorrectedSummary: %v", err)
		http.Error(w, "Une erreur est survenue. Veuillez réessayer.", http.StatusInternalServerError)
		return
	}

	examName := filepath.Base(target.ExamName.String)
	leftPagesName := strings.TrimSuffix(examName, filepath.Ext(examName)) + "_NOT.pdf"
	hasLeftPages, err := tools.MarkingArtifactExists(username, jobID, leftPagesName)
	if err != nil {
		log.Printf("From SuccessMarkingProcessingHandler -> inspect non-corrected PDF: %v", err)
		http.Error(w, "Une erreur est survenue. Veuillez réessayer.", http.StatusInternalServerError)
		return
	}

	dataPage := buildMarkingResultPageData(jobID, target, summary, reviewStatus, nonCorrected, hasLeftPages)
	if generation.Valid {
		dataPage.GenerationURL = markingGenerationURL(generation.Int64)
	}
	if r.URL.Query().Get("notice") == "artifacts_failed" && reviewStatus == db.MarkingReviewCompleted && !dataPage.Review.ArtifactsCurrent {
		dataPage.Alert = data.NoticeView{
			Title: "Actualisation des PDF impossible",
			Text:  "Les réponses sont enregistrées, mais les PDF n'ont pas pu être actualisés.",
		}
	}

	RenderSuccessProgressMarkingPage(w, dataPage)
}

func buildMarkingResultPageData(jobID int64, target db.GetMarkingArtifactsRegenerationTargetRow, summary db.GetMarkingReviewSummaryRow, reviewStatus db.MarkingReviewStatus, nonCorrected db.GetMarkingNonCorrectedSummaryRow, hasLeftPages bool) data.MarkingResultPageData {
	operation := "marking-" + strconv.FormatInt(jobID, 10)
	artifactURL := func(filename string) string {
		if filename == "" {
			return ""
		}
		return data.DefaultMarkingRoutes.ServePDF + "?operation=" + url.QueryEscape(operation) + "&file=" + url.QueryEscape(filepath.Base(filename))
	}
	current := target.ArtifactsRevision == target.ReviewRevision
	artifacts := data.MarkingArtifactLinksView{}
	hybridPending := target.ReviewPolicyVersion.Valid && reviewStatus == db.MarkingReviewPending
	if reviewStatus == db.MarkingReviewUnavailable || (!hybridPending && current) {
		artifacts.CorrectedPDFURL = artifactURL(target.ExamName.String)
		artifacts.MarkTablePDFURL = artifactURL(target.MarkTableName.String)
	}
	if hasLeftPages {
		leftPagesName := strings.TrimSuffix(filepath.Base(target.ExamName.String), filepath.Ext(target.ExamName.String)) + "_NOT.pdf"
		artifacts.NonCorrectedPDFURL = artifactURL(leftPagesName)
	}
	if reviewStatus == db.MarkingReviewCompleted && !current {
		artifacts.RegenerateURL = data.DefaultMarkingRoutes.ArtifactsRegenerate
	}

	notice := data.NoticeView{}
	switch reviewStatus {
	case db.MarkingReviewNoReviewNeeded:
		notice.Title = "Aucune réponse à vérifier"
		notice.Text = "Ce lot ne contient aucune réponse ambiguë."
	case db.MarkingReviewPending:
		notice.Title = strconv.FormatInt(summary.PendingCandidates, 10) + " réponses à vérifier"
		notice.Text = "Vérifiez les réponses ambiguës avant de considérer les PDF comme définitifs."
	case db.MarkingReviewCompleted:
		if current {
			notice.Title = "Toutes les réponses ont été vérifiées"
			notice.Text = "Les PDF correspondent aux réponses enregistrées."
		} else {
			notice.Title = "Les réponses sont enregistrées — les PDF doivent être actualisés"
			notice.Text = "Actualisation nécessaire avant de télécharger les PDF finaux."
		}
	case db.MarkingReviewUnavailable:
		notice.Title = "Revue assistée non disponible pour cette ancienne correction"
		notice.Text = "Les PDF historiques restent accessibles."
	}

	return data.MarkingResultPageData{
		Routes: data.DefaultDashboardRoutes, MarkingRoutes: data.DefaultMarkingRoutes,
		PageTitle: "Résultat de la correction", JobID: jobID, Notice: notice, Artifacts: artifacts,
		Review: data.MarkingReviewStatusView{
			Status: string(reviewStatus), TotalCandidates: summary.TotalCandidates,
			ReviewedCandidates: summary.ReviewedCandidates, PendingCandidates: summary.PendingCandidates,
			ArtifactsCurrent: current,
			ReviewURL:        data.DefaultMarkingRoutes.ReviewURL + "?job_id=" + url.QueryEscape(strconv.FormatInt(jobID, 10)),
			RevisitURL:       data.DefaultMarkingRoutes.ReviewURL + "?job_id=" + url.QueryEscape(strconv.FormatInt(jobID, 10)) + "&revisit=1",
		},
		NonCorrected: data.MarkingNonCorrectedSummaryView{
			Incomplete: nonCorrected.IncompleteCopies, Errors: nonCorrected.ErrorCopies,
			NotSeen: nonCorrected.NotSeenCopies,
			Total:   nonCorrected.IncompleteCopies + nonCorrected.ErrorCopies + nonCorrected.NotSeenCopies,
		},
	}
}

func ServeFullMarkingPdfHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From ServeFullMarkingPdfHandler -> tools.CheckRequest return not ok")
		return
	}

	if username == "" {
		log.Println("From ServeFullMarkingPdfHandler , no username")
		http.Error(w, "La requête est invalide ou incomplète.", http.StatusBadRequest)
		return
	}

	filename := r.URL.Query().Get("file")
	operation := r.URL.Query().Get("operation")
	if filename == "" || operation == "" {
		http.Error(w, "La demande est incomplète : le fichier est manquant.", http.StatusBadRequest)
		return
	}
	const prefix = "marking-"
	if !strings.HasPrefix(operation, prefix) {
		http.NotFound(w, r)
		return
	}
	jobID, err := strconv.ParseInt(strings.TrimPrefix(operation, prefix), 10, 64)
	if err != nil || jobID <= 0 || operation != prefix+strconv.FormatInt(jobID, 10) {
		http.NotFound(w, r)
		return
	}
	target, err := queries.GetMarkingArtifactsRegenerationTarget(r.Context(), db.GetMarkingArtifactsRegenerationTargetParams{MarkingJobID: jobID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("From ServeFullMarkingPdfHandler -> GetMarkingArtifactsRegenerationTarget: %v", err)
		http.Error(w, "Une erreur est survenue. Veuillez réessayer.", http.StatusInternalServerError)
		return
	}
	if (target.ReviewPolicyVersion.Valid && target.PendingCandidates > 0) || target.ArtifactsRevision != target.ReviewRevision {
		http.Error(w, "La vérification des réponses doit être terminée avant le téléchargement.", http.StatusConflict)
		return
	}

	downloadFilename := filename
	switch filepath.Base(filename) {
	case filepath.Base(target.ExamName.String):
		downloadFilename = "corrected.pdf"
		if target.SourcePdfFilename.Valid {
			downloadFilename = tools.MarkingArtifactFilename(target.SourcePdfFilename.String, tools.MarkingArtifactCorrected)
		}
	case filepath.Base(target.MarkTableName.String):
		downloadFilename = "marks.pdf"
		if target.SourcePdfFilename.Valid {
			downloadFilename = tools.MarkingArtifactFilename(target.SourcePdfFilename.String, tools.MarkingArtifactMarks)
		}
	default:
		tools.ServePdfNamed(username, operation, filename, w, r)
		return
	}
	tools.ServePdfDownloadNamed(username, operation, filename, downloadFilename, w, r)
}

/*
func ProcessingMarkingHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From ProcessingMarkingHandler-> tools.CheckRequest return not ok")
		return
	}

	file, err := tools.CheckPdfFile(r, 100<<20) // 100 Mo ou (50<<20) pour 50 Mo
	if err != nil {
		log.Printf("From ProcessingMarkingHandler -> CheckPdfFile : error : %v", err)
		errorMessage := url.QueryEscape("Fichier probablement trop volumineux ou le fichier n'est pas un pdf.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	tempDir, ok := tools.CreateUserTempDir(username)
	if !ok {
		log.Println("From ProcessingMarkingHandler -> CreateUserTempDir return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	if err := tools.ClearDir(tempDir); err != nil {
		log.Printf("From ProcessingMarkingHandler -> ClearDir return error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	if err := tools.SplitPdf(file, tempDir, "page-%d.pdf"); err != nil {
		log.Printf("From ProcessingMarkingHandler -> SplitPdf return error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	pages, err := tools.GetAllFiles(tempDir, "*.pdf")
	if err != nil {
		log.Printf("From ProcessingMarkingHandler -> GetAllFiles return error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	// commencer ici la go routine ?

	ctx := context.Background()

	var qrDatas []config.QrCodeInfo
	var qrNotDetected []string
	for _, page := range pages {
		pdf := filepath.Base(page)
		name := strings.TrimSuffix(pdf, filepath.Ext(page)) + ".png"
		png := tools.ConvertPdfToPng(tempDir, pdf, "")
		pngPath := filepath.Join(tempDir, png)
		data, err := tools.QrReader(pngPath)
		if err != nil {
			log.Printf("file : %s, qr not detected : error : %v \n", pngPath, err)
			qrNotDetected = append(qrNotDetected, png)
		} else {
			var info config.QrCodeInfo
			if err := json.Unmarshal([]byte(data), &info); err != nil {
				log.Printf("From ProcessingMarkingHandler -> Unmarshal return error : %v", err)
				http.Error(w, "Something went wrong !", http.StatusInternalServerError)
				return
			}
			info.PageName = name

			qrDatas = append(qrDatas, info)
		}
	}

	if err := tools.RemoveFiles(pages); err != nil {
		log.Printf("From ProcessingMarkingHandler -> RemoveFiles return error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	exams := tools.GroupQrCodes(qrDatas)

	var markExams []config.MarkExam
	var notMarkedExams []config.MarkExam

	for _, exam := range exams {
		markExam, err := tools.MarkingStudentExam(userID, username, tempDir, exam, ctx, queries)
		if err != nil {
			log.Printf("Error with MarkingStudentExam, %v", err)
			notMarkedExams = append(notMarkedExams, markExam)
		}
		if markExam.Status {
			markExams = append(markExams, markExam)
		}
	}

	fmt.Println("Done !")
}
*/
