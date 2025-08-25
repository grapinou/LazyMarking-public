package generateexams

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func GenerateExamsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
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
		http.Error(w, "DB error", http.StatusInternalServerError)
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

	classCodeName, err := queries.GetClassCodeNameByID(r.Context(), db.GetClassCodeNameByIDParams{
		ID:     exam.ClassCodeID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From GenerateExamsHandler -> GetClassCodeNameByID : DB error : %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	for _, stu := range students {
		student := config.StudentQCM{
			ID:        stu.ID,
			FirstName: stu.FirstName,
			LastName:  stu.LastName,
			ClassCodes: config.ClassCode{
				ID:   exam.ClassCodeID,
				Name: classCodeName,
			},
		}

		questions, err := tools.GetQCMQuestionsAnswers(userID, exam.QcmID, r, queries)
		if err == tools.ErrQuestionWithNoAnswer {
			log.Printf("From PreviewQCMHandler -> GetQCMQuestionsAnswers -> BuildQuestion : error : %v", err)
			errorMessage := url.QueryEscape("Il y a une question qui n'a pas de réponse. Il n'est pas possible de construire le qcm")
			http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
			return
		}
		if err != nil {
			log.Printf("From PreviewQCMHandler -> GetQCMQuestionsAnswers (-> BuildQuestion) : error : %v", err)
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}

		qcm := config.QCM{
			Name:      exam.Name,
			Student:   student,
			Questions: questions,
		}

		typstFilePath, ok := tools.TypstWriter(username, qcm, config.ExamQCM)
		if !ok {
			log.Println("From PreviewQuestionHandler -> tools.TypstWriter return not ok")
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}

		_, ok = tools.CompileTypst(typstFilePath)
		if !ok {
			log.Println("From PreviewQuestionHandler -> tools.CompileTypst return not ok")
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}

		_, ok = tools.ExportTypstToPNGs(typstFilePath)
		if !ok {
			log.Println("From PreviewQuestionHandler -> tools.ExportTypstToPNGs return not ok")
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.ExamURL, http.StatusSeeOther)
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

	typstFilePath, ok := tools.TypstWriterLandscapeAllContent(username, allContent)
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

	http.Redirect(w, r, data.DefaultGenerateExamRoutes.MiniQCMLandscape, http.StatusSeeOther)
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

	tools.ServePdf(username, config.MiniQCM, w)
}
