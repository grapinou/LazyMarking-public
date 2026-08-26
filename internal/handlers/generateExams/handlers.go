package generateexams

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
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

func GenerateExamsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries, appCtx context.Context, backgroundJobs *sync.WaitGroup) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
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
		log.Printf("From GenerateExamsHandler -> GetExamByID DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
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
	tempDir, ok := tools.CreateOperationTempDir(username, operation)
	if !ok {
		if err := failExamGeneration(userID, examGeneratedID, r.Context(), queries); err != nil {
			log.Printf("From GenerateExamsHandler -> fail generation after workspace error: %v", err)
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
		if failErr := failExamGeneration(userID, examGeneratedID, r.Context(), queries); failErr != nil {
			log.Printf("From GenerateExamsHandler -> fail generation after class code error: %v", failErr)
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

				// Contexte par étudiant
				ctx, cancel := context.WithTimeout(appCtx, 60*time.Second)
				defer cancel()

				_, err := tools.BuildQcmStudentCtx(
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
			if err := failExamGeneration(userID, examGeneratedID, appCtx, queries); err != nil {
				log.Printf("From GenerateExamsHandler -> fail generation: %v", err)
			}
			return
		}

		pdfFiles, err := tools.GetAllFiles(tempDir, "*.pdf")
		if err != nil {
			log.Printf("From GenerateExamsHandler -> tools.GetAllFiles pdf: %v", err)
			if failErr := failExamGeneration(userID, examGeneratedID, appCtx, queries); failErr != nil {
				log.Printf("From GenerateExamsHandler -> fail generation after listing PDFs: %v", failErr)
			}
			return
		}

		finalPDFName := examGenerationPDFName(username, exam.Name, classCodeName)
		if err := tools.MergePdf(pdfFiles, filepath.Join(tempDir, finalPDFName)); err != nil {
			log.Printf("From GenerateExamsHandler -> tools.MergePdf: %v", err)
			if failErr := failExamGeneration(userID, examGeneratedID, appCtx, queries); failErr != nil {
				log.Printf("From GenerateExamsHandler -> fail generation after PDF merge: %v", failErr)
			}
			return
		}

		cleanupExamGenerationFiles(tempDir, pdfFiles)

		if err := queries.UpdateExamGenerated(appCtx, db.UpdateExamGeneratedParams{
			Status: "success",
			ID:     examGeneratedID,
			UserID: userID,
		}); err != nil {
			log.Printf("From GenerateExamsHandler -> queries.UpdateExamGenerated success: %v", err)
			if failErr := failExamGeneration(userID, examGeneratedID, appCtx, queries); failErr != nil {
				log.Printf("From GenerateExamsHandler -> fail generation after success update: %v", failErr)
			}
		}
	}()

	params := "?exam_generated_id=" + url.QueryEscape(strconv.FormatInt(examGeneratedID, 10))
	processingStudentURL := data.DefaultGenerateExamRoutes.ProcessingStudents + params
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
		log.Printf("From GetExamProgressHandler -> GetExamStatus : error : %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	if examStatus == "failed" {
		log.Println("From GetExamProgressHandler -> exam status failed")
		errorMessage := url.QueryEscape("Erreur lors de la génération du qcm, contacter admin")
		if err := queries.DeleteExamGenerated(r.Context(), db.DeleteExamGeneratedParams{
			ID:     examGeneratedID,
			UserID: userID,
		}); err != nil {
			log.Printf("From GetExamProgressHandler -> DeleteExamGenerated : error : %v", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	if examStatus == "success" {
		names, err := queries.GetExamNameAndClassCodeName(r.Context(), db.GetExamNameAndClassCodeNameParams{
			ID:     examGeneratedID,
			UserID: userID,
		})
		if err != nil {
			log.Printf("From GetExamProgressHandler -> queries.GetExamNameAndClassCodeName: DB error : %v", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}

		operation := "exam-" + strconv.FormatInt(examGeneratedID, 10)
		name := examGenerationPDFName(username, names.ExamName, names.ClassName)
		pdfURL := data.DefaultGenerateExamRoutes.PdfExam + "?operation=" + url.QueryEscape(operation) + "&file=" + url.QueryEscape(name)
		dataPage := data.GenerateExamPageData{
			Routes:             data.DefaultDashboardRoutes,
			GenerateExamRoutes: data.DefaultGenerateExamRoutes,
			PageTitle:          "Success Processing",
			ExtraData: map[string]any{
				"Status": "success",
				"PdfURL": pdfURL,
			},
		}

		RenderSuccessProcessing(w, dataPage)
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

	dataPage := data.GenerateExamPageData{
		Routes:             data.DefaultDashboardRoutes,
		GenerateExamRoutes: data.DefaultGenerateExamRoutes,
		PageTitle:          "Processing Students",
		ExtraData: map[string]any{
			"ExamGeneratedID": examGenIDStr,
			"Processed":       row.ProcessedStudents,
			"Total":           row.TotalStudents,
			"Status":          examStatus,
		},
	}

	RenderProcessingStudentsPage(w, dataPage)
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
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
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
		log.Printf("From GenerateMiniPDFHandler -> GetExamByID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
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

	const maxConcurrentStudents = 5
	studentSemaphore := make(chan struct{}, maxConcurrentStudents)

	var wg sync.WaitGroup
	results := make(chan string, len(students))
	errs := make(chan error, len(students))

	for _, stu := range students {
		wg.Add(1)
		go func(stu db.Student) {
			defer wg.Done()

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
			content := tools.TypstLandscapeContent(qcm)
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

	operation := "mini-" + uuid.NewString()
	tempDir, ok := tools.CreateOperationTempDir(username, operation)
	if !ok {
		http.Error(w, "Unable to create generation workspace", http.StatusInternalServerError)
		return
	}
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

	http.Redirect(w, r, data.DefaultGenerateExamRoutes.MiniQCMLandscape+"?operation="+url.QueryEscape(operation), http.StatusSeeOther)
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
