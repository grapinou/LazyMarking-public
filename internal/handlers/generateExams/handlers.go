package generateexams

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

var createExamGenerationWorkspace = tools.CreateOperationTempDir
var buildQCMStudentForGeneration = tools.BuildQcmStudentCtx
var resolveExamGenerationPDFName = tools.ResolveExamGenerationPDFName

func GenerateExamsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries, appCtx context.Context, backgroundJobs *sync.WaitGroup) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From GenerateExamsHandler -> tools.CheckRequest return not ok")
		return
	}

	examIDStr := r.URL.Query().Get("exam_id")
	if examIDStr == "" {
		log.Println("From GenerateExamsHandler : no exam id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	examID, err := strconv.ParseInt(examIDStr, 10, 64)
	if err != nil {
		log.Printf("From GenerateExamsHandler -> strconv.ParseInt invalid question id parameter, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	exam, err := queries.GetExamByID(r.Context(), db.GetExamByIDParams{
		ID:     examID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "GenerateExamsHandler GetExamByID")
		return
	}

	students, err := tools.GetAllStudentsFromClassCode(userID, exam.ClassCodeID, r, queries)
	if err == tools.ErrClassCodeWithNoStudents {
		log.Printf("From GenerateExamsHandler -> GetAllStudentsFromClassCode : error : %v", err)
		errorMessage := url.QueryEscape("La classe sélectionnée ne contient aucun élève, pas possible de faire l'examen.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	} else if err != nil {
		log.Printf("From GenerateExamsHandler -> GetAllStudentsFromClassCode : DB error : %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	if !validateExamQCMHasQuestions(w, r, queries, userID, exam.QcmID) {
		return
	}

	invalidNames := tools.CheckStudentsNames(students)
	if len(invalidNames) != 0 {
		log.Println("From GenerateExamsHandler -> names with \" detected")
		msg := strings.Join(invalidNames, ", ")
		errMsg := fmt.Sprintf("Les noms suivants contiennent un \". Corriger les ! %s", msg)
		errorMessage := url.QueryEscape(errMsg)
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	examGeneratedID, err := queries.CreateExamGenerated(r.Context(), db.CreateExamGeneratedParams{
		ExamID:        examID,
		TotalStudents: int64(len(students)),
		UserID:        userID,
	})
	if err != nil {
		log.Printf("From GenerateExamsHandler -> CreateExamGenerated DB error: %v", err)
		errorMessage := url.QueryEscape("L'examen a déjà été généré. Vous pouvez le corriger ou le supprimer.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	operation := "exam-" + strconv.FormatInt(examGeneratedID, 10)
	tempDir, ok := createExamGenerationWorkspace(username, operation)
	if !ok {
		if err := cleanupFailedExamGeneration(userID, examGeneratedID, username, r.Context(), queries); err != nil {
			log.Printf("From GenerateExamsHandler -> cleanup generation after workspace error: %v", err)
		}
		http.Error(w, "Unable to create generation workspace", http.StatusInternalServerError)
		return
	}

	classCodeName, err := queries.GetClassCodeNameByID(r.Context(), db.GetClassCodeNameByIDParams{
		ID:     exam.ClassCodeID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From GenerateExamsHandler -> GetClassCodeNameByID : DB error : %v", err)
		if cleanupErr := cleanupFailedExamGeneration(userID, examGeneratedID, username, r.Context(), queries); cleanupErr != nil {
			log.Printf("From GenerateExamsHandler -> cleanup generation after class code error: %v", cleanupErr)
		}
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	// 🚀 Lancer la génération en arrière-plan :
	// ⚡ Sémaphore pour limiter la concurrence
	sem := make(chan struct{}, 5)

	// 📦 Canal pour collecter les erreurs
	errs := make(chan error, len(students))

	backgroundJobs.Add(1)
	go func() {
		defer backgroundJobs.Done()
		generationSucceeded := false
		cleanupAttempted := false
		cleanupFailure := func() {
			if cleanupAttempted {
				return
			}
			cleanupAttempted = true
			if err := cleanupFailedExamGeneration(userID, examGeneratedID, username, appCtx, queries); err != nil {
				log.Printf("From GenerateExamsHandler -> cleanup failed generation: %v", err)
			}
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("From GenerateExamsHandler -> recovered generation panic: %v", recovered)
			}
			if !generationSucceeded {
				cleanupFailure()
			}
		}()
		// start := time.Now()
		var wg sync.WaitGroup

		generationCanceled := false
	studentLoop:
		for _, stu := range students {
			if appCtx.Err() != nil {
				generationCanceled = true
				break
			}
			select {
			case sem <- struct{}{}: // prendre une place
			case <-appCtx.Done():
				generationCanceled = true
				break studentLoop
			}
			wg.Add(1)
			go func(stu db.Student) {
				defer wg.Done()
				defer func() { <-sem }() // libérer la place
				defer func() {
					if recovered := recover(); recovered != nil {
						errs <- fmt.Errorf("student %d worker panic: %v", stu.ID, recovered)
					}
				}()

				// Contexte par étudiant
				ctx, cancel := context.WithTimeout(appCtx, 60*time.Second)
				defer cancel()

				_, err := buildQCMStudentForGeneration(
					stu,
					exam,
					examGeneratedID,
					userID,
					tempDir,
					username,
					classCodeName,
					ctx,
					queries,
				)
				if err != nil {
					// envoyer l’erreur dans le canal
					errs <- fmt.Errorf("student %d: %w", stu.ID, err)
					return
				}

				// log.Printf("✅ QCM généré pour %s %s", stu.FirstName, stu.LastName)
			}(stu)
		}

		wg.Wait()
		close(errs) // ⚠️ fermer le canal une fois que toutes les goroutines sont terminées

		// elapsed := time.Since(start)
		// log.Printf("🎉 Génération terminée en %s", elapsed)

		// 🔍 Lire toutes les erreurs collectées
		errorsOccured := generationCanceled || appCtx.Err() != nil
		for err := range errs {
			log.Printf("From GenerateExamsHandler -> tools.BuildQcmStudentCtx error : %v", err)
			errorsOccured = true
		}

		if errorsOccured {
			return
		}

		pdfFiles, err := tools.GetAllFiles(tempDir, "*.pdf")
		if err != nil {
			log.Printf("From GenerateExamsHandler -> tools.GetAllFiles pdf: %v", err)
			return
		}

		finalPDFName := examGenerationPDFName(username, exam.Name, classCodeName)
		if err := tools.MergePdf(pdfFiles, filepath.Join(tempDir, finalPDFName)); err != nil {
			log.Printf("From GenerateExamsHandler -> tools.MergePdf: %v", err)
			return
		}

		cleanupExamGenerationFiles(tempDir, pdfFiles)

		if err := completeExamGeneration(userID, examGeneratedID, appCtx, queries); err != nil {
			log.Printf("From GenerateExamsHandler -> completeExamGeneration: %v", err)
			return
		}
		generationSucceeded = true
	}()

	progressQuery := url.Values{
		"exam_id":            {strconv.FormatInt(examID, 10)},
		"generation_started": {"1"},
	}
	processingStudentURL := examGenerationProgressURL(examGeneratedID, progressQuery)
	http.Redirect(w, r, processingStudentURL, http.StatusSeeOther)
}

func GetExamProgressPageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From GetExamProgressHandler -> tools.CheckRequest return not ok")
		return
	}

	examGenIDStr := r.URL.Query().Get("exam_generated_id")
	if examGenIDStr == "" {
		log.Println("From GetExamProgressHandler -> no exam generated id")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	examGeneratedID, err := strconv.ParseInt(examGenIDStr, 10, 64)
	if err != nil {
		log.Printf("From GetExamProgressHandler-> strconv.ParseInt invalid examGeneratedID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	examStatus, err := queries.GetExamStatus(r.Context(), db.GetExamStatusParams{
		ID:     examGeneratedID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if handleCleanedFailedGenerationPoll(w, r, queries, userID) {
				return
			}
			http.NotFound(w, r)
			return
		}
		log.Printf("From GetExamProgressHandler -> GetExamStatus : error : %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	if examStatus == "failed" {
		log.Println("From GetExamProgressHandler -> exam status failed")
		redirectFailedExamGeneration(w, r)
		return
	}

	if examStatus == "success" {
		names, err := queries.GetExamNameAndClassCodeName(r.Context(), db.GetExamNameAndClassCodeNameParams{
			ID:     examGeneratedID,
			UserID: userID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			log.Printf("From GetExamProgressHandler -> queries.GetExamNameAndClassCodeName: DB error : %v", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		pdfName, err := resolveExamGenerationPDFName(username, examGeneratedID)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				RenderUnavailableExamPDF(w, buildExamGenerationUnavailablePageData(examGeneratedID, names))
				return
			}
			log.Printf("From GetExamProgressHandler -> resolve generation PDF: %v", err)
			http.Error(w, "Unable to access generated PDF", http.StatusInternalServerError)
			return
		}

		RenderSuccessProcessing(w, buildExamGenerationSuccessPageData(examGeneratedID, names, pdfName))
		return
	}

	row, err := queries.GetExamGeneratedProgress(r.Context(), db.GetExamGeneratedProgressParams{
		ID:     examGeneratedID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From GetExamProgressHandler -> queries.GetExamGeneratedProgress : DB error : %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	RenderProcessingStudentsPage(w, buildExamGenerationProgressPageData(
		examGeneratedID,
		examStatus,
		row,
		examGenerationProgressURL(examGeneratedID, r.URL.Query()),
	))
}

func examGenerationProgressURL(generationID int64, query url.Values) string {
	params := url.Values{}
	for key, values := range query {
		params[key] = append([]string(nil), values...)
	}
	params.Set("exam_generated_id", strconv.FormatInt(generationID, 10))
	return data.DefaultGenerateExamRoutes.ProcessingStudents + "?" + params.Encode()
}

func examGenerationCopiesURL(generationID int64, pdfName string) string {
	operation := "exam-" + strconv.FormatInt(generationID, 10)
	params := url.Values{"operation": {operation}, "file": {pdfName}}
	return data.DefaultGenerateExamRoutes.PdfExam + "?" + params.Encode()
}

func buildExamGenerationProgressPageData(generationID int64, status string, progress db.GetExamGeneratedProgressRow, progressURL string) data.GenerateExamPageData {
	return data.GenerateExamPageData{
		PageTitle: "Processing Students",
		Routes:    data.DefaultDashboardRoutes,
		Context:   data.ExamGenerationContext{GenerationID: generationID},
		Progress: data.ExamGenerationProgress{
			Status:            status,
			ProcessedStudents: progress.ProcessedStudents,
			TotalStudents:     progress.TotalStudents,
			ProgressURL:       progressURL,
			ExamsURL:          data.DefaultDashboardRoutes.ExamURL,
		},
	}
}

func buildExamGenerationSuccessPageData(generationID int64, names db.GetExamNameAndClassCodeNameRow, pdfName string) data.GenerateExamPageData {
	return data.GenerateExamPageData{
		PageTitle: "Success Processing",
		Routes:    data.DefaultDashboardRoutes,
		Context: data.ExamGenerationContext{
			GenerationID: generationID,
			ExamName:     names.ExamName,
			ClassName:    names.ClassName,
		},
		Success: data.ExamGenerationSuccessData{
			Status:    "success",
			CopiesURL: examGenerationCopiesURL(generationID, pdfName),
			ExamsURL:  data.DefaultDashboardRoutes.ExamURL,
		},
	}
}

func buildExamGenerationUnavailablePageData(generationID int64, names db.GetExamNameAndClassCodeNameRow) data.GenerateExamPageData {
	return data.GenerateExamPageData{
		PageTitle: "Copies indisponibles",
		Routes:    data.DefaultDashboardRoutes,
		Context: data.ExamGenerationContext{
			GenerationID: generationID,
			ExamName:     names.ExamName,
			ClassName:    names.ClassName,
		},
		Unavailable: data.ExamGenerationUnavailableData{
			ExamsURL: data.DefaultDashboardRoutes.ExamURL,
		},
	}
}

func ServeFullPdfExamHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From ServeFullPdfExamHandler -> tools.CheckRequest return not ok")
		return
	}

	if username == "" {
		log.Println("From ServeFullPdfExamHandler, no username")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	filename := r.URL.Query().Get("file")
	operation := r.URL.Query().Get("operation")
	if filename == "" || operation == "" {
		http.Error(w, "Missing file parameter", http.StatusBadRequest)
		return
	}

	tools.ServePdfNamed(username, operation, filename, w, r)
}

func GenerateMiniPDFHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From GenerateMiniPDFHandler -> tools.CheckRequest return not ok")
		return
	}

	examIDStr := r.URL.Query().Get("exam_id")
	if examIDStr == "" {
		log.Println("From GenerateMiniPDFHandler : no exam id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	examID, err := strconv.ParseInt(examIDStr, 10, 64)
	if err != nil {
		log.Printf("From GenerateMiniPDFHandler -> strconv.ParseInt invalid question id parameter, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	exam, err := queries.GetExamByID(r.Context(), db.GetExamByIDParams{
		ID:     examID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "GenerateMiniPDFHandler GetExamByID")
		return
	}

	students, err := tools.GetAllStudentsFromClassCode(userID, exam.ClassCodeID, r, queries)
	if err == tools.ErrClassCodeWithNoStudents {
		log.Printf("From GenerateMiniPDFHandler -> GetAllStudentsFromClassCode : error : %v", err)
		errorMessage := url.QueryEscape("La classe sélectionnée ne contient aucun élève, pas possible de faire l'examen.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	} else if err != nil {
		log.Printf("From GenerateMiniPDFHandler -> GetAllStudentsFromClassCode : DB error : %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	if !validateExamQCMHasQuestions(w, r, queries, userID, exam.QcmID) {
		return
	}

	const maxConcurrentStudents = 5
	studentSemaphore := make(chan struct{}, maxConcurrentStudents)

	var wg sync.WaitGroup
	results := make(chan string, len(students))
	errs := make(chan error, len(students))

	for _, stu := range students {
		wg.Add(1)
		go func(stu db.Student) {
			defer wg.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					errs <- fmt.Errorf("student %d mini generation panic: %v", stu.ID, recovered)
				}
			}()

			// Limite locale par utilisateur : max 5 étudiants en parallèle
			studentSemaphore <- struct{}{}
			defer func() { <-studentSemaphore }()

			// Récupérer les questions/réponses avec limite DB globale
			questions, err := tools.GetQCMQuestionsAnswers(userID, exam.QcmID, r, queries)
			if err != nil {
				errs <- fmt.Errorf("student %v: %w", stu.ID, err)
				return
			}

			// Générer le contenu Typst
			qcm := config.QCM{Questions: questions}
			content, err := tools.TypstLandscapeContent(qcm)
			if err != nil {
				errs <- fmt.Errorf("student %v Typst content: %w", stu.ID, err)
				return
			}
			results <- content
		}(stu)
	}

	wg.Wait()
	close(results)
	close(errs)

	if len(errs) > 0 {
		for e := range errs {
			log.Printf("From GenerateMiniPDFHandler -> student processing error: %v", e)
		}
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	var allContent []string
	for c := range results {
		allContent = append(allContent, c)
	}

	if err := tools.PurgeExpiredUserEphemeralWorkspaces(username, time.Now()); err != nil {
		log.Printf("From GenerateMiniPDFHandler -> purge stale mini workspaces: %v", err)
	}
	operation := "mini-" + uuid.NewString()
	tempDir, ok := tools.CreateOperationTempDir(username, operation)
	if !ok {
		http.Error(w, "Unable to create generation workspace", http.StatusInternalServerError)
		return
	}
	keepWorkspace := false
	defer func() {
		if keepWorkspace {
			return
		}
		if err := tools.RemoveOperationTempDir(username, operation); err != nil {
			log.Printf("From GenerateMiniPDFHandler -> cleanup failed mini workspace: %v", err)
		}
	}()
	typstFilePath, ok := tools.TypstWriterLandscapeAllContent(tempDir, username, allContent)
	if !ok {
		log.Println("From GenerateMiniPDFHandler -> tools.TypstWriterLandscapeAllContent return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	_, ok = tools.CompileTypst(typstFilePath)
	if !ok {
		log.Println("From GenerateMiniPDFHandler -> tools.CompileTypst return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	keepWorkspace = true
	http.Redirect(w, r, data.DefaultGenerateExamRoutes.MiniQCMLandscape+"?operation="+url.QueryEscape(operation), http.StatusSeeOther)
}

func validateExamQCMHasQuestions(w http.ResponseWriter, r *http.Request, queries *db.Queries, userID, qcmID int64) bool {
	questionIDs, err := queries.GetQCMQuestionsIDs(r.Context(), db.GetQCMQuestionsIDsParams{
		UserID: userID,
		QcmID:  qcmID,
	})
	if err != nil {
		log.Printf("validateExamQCMHasQuestions -> GetQCMQuestionsIDs: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return false
	}
	if len(questionIDs) == 0 {
		errorMessage := url.QueryEscape("Ajoutez au moins une question au QCM avant de générer l'évaluation.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return false
	}
	return true
}

func ServeMiniPDFHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From ServeMiniPDFHandler -> tools.CheckRequest return not ok")
		return
	}

	if username == "" {
		log.Println("From ServeMiniPDFHandler, no username")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	operation := r.URL.Query().Get("operation")
	if operation == "" {
		http.Error(w, "Missing operation parameter", http.StatusBadRequest)
		return
	}
	typstName := username + string(config.MiniQCM)
	filename := strings.TrimSuffix(typstName, filepath.Ext(typstName)) + ".pdf"
	tools.ServePdfNamed(username, operation, filename, w, r)
}
