package tools

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func BuildQcmStudentCtx(stu db.Student, exam db.Exam, examGeneratedID, userID int64, username, classCodeName string, ctx context.Context, queries *db.Queries) (config.QCM, error) {
	var qcm config.QCM
	student := config.StudentQCM{
		ID:        stu.ID,
		FirstName: stu.FirstName,
		LastName:  stu.LastName,
		ClassCodes: config.ClassCode{
			ID:   exam.ClassCodeID,
			Name: classCodeName,
		},
	}

	studentExamID, err := queries.CreateStudentExam(ctx, db.CreateStudentExamParams{
		ExamGeneratedID: examGeneratedID,
		StudentID:       student.ID,
		UserID:          userID,
	})
	if err != nil {
		log.Printf("CreateStudentExam : DB error : %v", err)
		return qcm, err
	}

	questions, err := GetQCMQuestionsAnswersCtx(userID, exam.QcmID, ctx, queries)
	if err == ErrQuestionWithNoAnswer {
		log.Printf("GetQCMQuestionsAnswers -> BuildQuestion : error : %v", err)
		return qcm, ErrQuestionWithNoAnswer
	}
	if err != nil {
		log.Printf("GetQCMQuestionsAnswers (-> BuildQuestion) : error : %v", err)
		return qcm, err
	}

	qcm = config.QCM{
		Name:      exam.Name,
		Student:   student,
		Questions: questions,
	}

	typstFilePath, ok := TypstWriter(username, qcm, config.ExamQCM)
	if !ok {
		log.Println("TypstWriter return not ok")
		return qcm, errors.New(" -> TypstWriter return not ok")
	}

	pages, ok := ExportTypstToPNGs(typstFilePath)
	if !ok {
		log.Println("ExportTypstToPNGs return not ok")
		return qcm, errors.New(" -> ExportTypstToPNGs return not ok")
	}

	var sortedQuestions []config.CircleValidated // pour stocker l'ensemble des questions de toutes les pages
	var sortedAnswers []config.CircleValidated   // pour stocker l'ensemble des réponses de toutes les pages
	var pageTot int64
	var pdfNames []string
	var temp string

	for _, page := range pages {

		// on commence par mettre un qr code avec les infos qu'il faut sur chaque page
		tempDir, pageName := filepath.Split(page)
		temp = tempDir

		pageNumber, total, ok := ExtractPageNumber(pageName)
		pageTot = int64(total)
		if !ok {
			log.Println("ExtractPageNumber return not ok")
			return qcm, errors.New(" -> ExtractPageNumber return not ok")
		}

		qrCodeInfo := config.QrCodeInfo{
			StudentExamID: studentExamID,
			PageExam:      pageNumber,
		}

		qrName, ok := QrCodeMaker(tempDir, pageName, qrCodeInfo)
		if !ok {
			log.Println("QrCodeMaker return not ok")
			return qcm, errors.New(" -> QrCodeMaker return not ok")
		}

		imgName, ok := PasteQrCodeOnPage(tempDir, qrName, pageName)
		if !ok {
			log.Println("PasteQrCodeOnPage return not ok")
			return qcm, errors.New(" -> PasteQrCodeOnPage return not ok")
		}

		// chaque png page est regardé pour voir ce qu'il y a dessus

		// on commence par les cercles de questions
		circles, ok := CircleDetection(tempDir, imgName) // circles représentent les cercles des questions
		if !ok {
			log.Println("CircleDetection return not ok")
			return qcm, errors.New(" -> CircleDetection return not ok")
		}

		// attention, une page peut être sans rond de questions !
		lenCircles := len(circles)
		if lenCircles > 0 {
			sortedQuestions = append(sortedQuestions, circles...)
		}

		// détection des cercles de réponses
		// si la page ne contient que des réponses

		if lenCircles == 0 {

			qrPostion := 415
			bottomPostion := 3390
			answers, ok := CircleDetectionAnswer(tempDir, imgName, qrPostion, bottomPostion)
			if !ok {
				log.Println("CircleDetectionAnswerreturn not ok : Between qrcode and first question")
				return qcm, errors.New(" -> CircleDetectionAnswerreturn not ok : Between qrcode and first question")
			}
			if len(answers) != 0 {
				sortedAnswers = append(sortedAnswers, answers...)
			}
		} else {

			// détection entre qrcode et première question
			qrPostion := 415
			answers, ok := CircleDetectionAnswer(tempDir, imgName, qrPostion, circles[0].Position.Y-circles[0].Radius)
			if !ok {
				log.Println("CircleDetectionAnswerreturn not ok : Between qrcode and first question")
				return qcm, errors.New(" -> CircleDetectionAnswerreturn not ok : Between qrcode and first question")
			}
			if len(answers) != 0 {
				sortedAnswers = append(sortedAnswers, answers...)
			}

			// détection entre les questions
			nbrQuestions := len(circles)
			if nbrQuestions > 1 {
				// ici on s'arrête à l'avant dernière question
				for i := 0; i < nbrQuestions-1; i++ {
					answers, ok = CircleDetectionAnswer(tempDir, imgName,
						circles[i].Position.Y+circles[i].Radius,
						circles[i+1].Position.Y-circles[i+1].Radius)
					if !ok || len(answers) == 0 {
						log.Println("CircleDetectionAnswerreturn not ok or no answers detected between questions")
						return qcm, errors.New(" -> CircleDetectionAnswerreturn not ok or no answers detected between questions")
					}
					sortedAnswers = append(sortedAnswers, answers...)
				}
			}

			// détection entre la dernière question et le bas de la page
			bottomPostion := 3390
			answers, ok = CircleDetectionAnswer(tempDir, imgName, circles[nbrQuestions-1].Position.Y+circles[nbrQuestions-1].Radius, bottomPostion)
			if !ok {
				log.Println("CircleDetectionAnswerreturn not ok : at bottom")
				return qcm, errors.New(" -> CircleDetectionAnswerreturn not ok : at bottom")
			}
			if len(answers) != 0 {
				sortedAnswers = append(sortedAnswers, answers...)
			}
		}

		// entrer en db de la page
		pageContent := config.PageContent{
			Questions: circles,
			Answers:   sortedAnswers,
		}
		// Sérialiser en JSON
		pageContentJSON, err := json.Marshal(pageContent)
		if err != nil {
			log.Printf("json.Marshal(pageContent) error : %v", err)
			return qcm, err
		}

		if err = queries.CreateStudentExamPageContent(ctx, db.CreateStudentExamPageContentParams{
			StudentExamID: studentExamID,
			Page:          int64(pageNumber),
			Content:       string(pageContentJSON),
			UserID:        userID,
		}); err != nil {
			log.Printf("From BuildQcmStudentCtx -> CreateStudentExamPageContent DB error : %v", err)
			return qcm, err
		}

		// création du pdf
		pdf := ConvertPngTopdf(tempDir, imgName)
		pdfPath := filepath.Join(tempDir, pdf)
		pdfNames = append(pdfNames, pdfPath)

		/*
			// pour voir si la détection est correcte (fonctionne pour une page, si plusieurs, dessine tous sur les autre car boucle parcours de nouvaux tous ce qui a été détecté !)
			// test sur les png de bases
			DrawCircleOnQcm(tempDir, imgName, "sur_png_", sortedQuestions, sortedAnswers)
			// making pdf file
			pdfName := ConvertPngTopdf(tempDir, imgName)
			// making pdf to png
			pdfToPngName := ConvertPdfToPng(tempDir, pdfName, "png_from_pdf_")

			// homography
			homoName := Homography(tempDir, pdfToPngName, imgName)
			DrawCircleOnQcm(tempDir, homoName, "sur_homo_", sortedQuestions, sortedAnswers)

			// homography alpha chan
			homoNameAlpha := HomographyWithAlpha(tempDir, pdfToPngName, imgName)
			DrawCircleOnQcm(tempDir, homoNameAlpha, "sur_homo_alpha_", sortedQuestions, sortedAnswers)

			// test sur les png des pdf
			DrawCircleOnQcm(tempDir, pdfToPngName, "sur_png_from_pdf_", sortedQuestions, sortedAnswers)
		*/

	}

	// vérification du bon nombre détecté de questions et de réponses détectées
	totQuestions := len(qcm.Questions)
	var totAnswers int
	for _, question := range qcm.Questions {
		totAnswers += len(question.Answers)
	}

	if totQuestions != len(sortedQuestions) || totAnswers != len(sortedAnswers) {
		log.Println("Number of questions or answers not matching numbers questions or answers detected")
		return qcm, errors.New(" -> Number of questions or answers not matching numbers questions or answers detected")
	}

	// fusion des pages du pdf en un seul pdf
	if len(pdfNames) > 1 {
		name := "merge_" + qcm.Student.FirstName + "_" + qcm.Student.LastName + "_" + qcm.Name + ".pdf"
		mergePath := filepath.Join(temp, name)
		if err := MergePdf(pdfNames, mergePath); err != nil {
			return qcm, err
		}

		if err := RemoveFiles(pdfNames); err != nil {
			return qcm, err
		}

	}

	// on a toutes les infos du qcm
	currentAnswer := 0
	for i := range qcm.Questions {
		qcm.Questions[i].Circle = sortedQuestions[i]

		n := len(qcm.Questions[i].Answers) // nombre de réponses de cette question
		if currentAnswer+n > len(sortedAnswers) {
			return qcm, errors.New("not enough answers in sortedAnswers")
		}

		for j := 0; j < n; j++ {
			qcm.Questions[i].Answers[j].Circle = sortedAnswers[currentAnswer]
			currentAnswer++
		}
	}

	// Sérialiser en JSON
	qcmJSON, err := json.Marshal(qcm)
	if err != nil {
		log.Printf("json.Marshal(qcm) error : %v", err)
		return qcm, err
	}

	err = queries.CreateStudentExamContent(ctx, db.CreateStudentExamContentParams{
		StudentExamID: studentExamID,
		PageTot:       pageTot,
		Content:       string(qcmJSON),
		UserID:        userID,
	})
	if err != nil {
		log.Printf("From BuildQcmStudentCtx -> CreateStudentExamContent DB error : %v", err)
		return qcm, err
	}

	if err := queries.UpdateExamGeneratedProcessedStudent(ctx, db.UpdateExamGeneratedProcessedStudentParams{
		ID:     examGeneratedID,
		UserID: userID,
	}); err != nil {
		log.Printf("From BuildQcmStudentCtx -> UpdateExamGeneratedProcessedStudent DB error : %v", err)
		return qcm, err
	}

	return qcm, nil
}
