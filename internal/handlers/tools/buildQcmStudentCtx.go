package tools

import (
	"context"
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
		return qcm, errors.New("from GenerateExamsHandler -> TypstWriter return not ok")
	}

	pages, ok := ExportTypstToPNGs(typstFilePath)
	if !ok {
		log.Println("ExportTypstToPNGs return not ok")
		return qcm, errors.New("from GenerateExamsHandler -> ExportTypstToPNGs return not ok")
	}

	var sortedQuestions []config.CircleValidated // pour stocker l'ensemble des questions de toutes les pages
	var sortedAnswers [][]config.CircleValidated // pour stocker l'ensemble des réponses de toutes les pages
	for _, page := range pages {

		tempDir, pageName := filepath.Split(page)

		pageNumber, _, ok := ExtractPageNumber(pageName)
		if !ok {
			log.Println("ExtractPageNumber return not ok")
			return qcm, errors.New("from GenerateExamsHandler -> ExtractPageNumber return not ok")
		}

		qrCodeInfo := config.QrCodeInfo{
			StudentExamID: studentExamID,
			PageExam:      pageNumber,
		}

		qrName, ok := QrCodeMaker(tempDir, qrCodeInfo)
		if !ok {
			log.Println("QrCodeMaker return not ok")
			return qcm, errors.New("from GenerateExamsHandler -> QrCodeMaker return not ok")
		}

		imgName, ok := PasteQrCodeOnPage(tempDir, qrName, pageName)
		if !ok {
			log.Println("PasteQrCodeOnPage return not ok")
			return qcm, errors.New("from GenerateExamsHandler -> PasteQrCodeOnPage return not ok")
		}
		circles, ok := CircleDetection(tempDir, imgName)
		if !ok {
			log.Println("CircleDetection return not ok")
			return qcm, errors.New("from GenerateExamsHandler -> CircleDetection return not ok")
		}

		sortedQuestions = append(sortedQuestions, circles...)

		// détection entre qrcode et première question
		qrPostion := 415
		answers, ok := CircleDetectionAnswer(tempDir, imgName, qrPostion, circles[0].Position.Y-circles[0].Radius)
		if !ok {
			log.Println("CircleDetectionAnswerreturn not ok")
			return qcm, errors.New("from GenerateExamsHandler -> CircleDetectionAnswerreturn not ok")
		}
		if len(answers) != 0 {
			sortedAnswers = append(sortedAnswers, answers)
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
					return qcm, errors.New("from GenerateExamsHandler -> CircleDetectionAnswerreturn not ok or no answers detected between questions")
				}
				sortedAnswers = append(sortedAnswers, answers)
			}
		}

		// détection entre la dernière question et le bas de la page
		bottomPostion := 3390
		answers, ok = CircleDetectionAnswer(tempDir, imgName, circles[nbrQuestions-1].Position.Y+circles[nbrQuestions-1].Radius, bottomPostion)
		if !ok {
			log.Println("CircleDetectionAnswerreturn not ok")
			return qcm, errors.New("from GenerateExamsHandler -> CircleDetectionAnswerreturn not ok")
		}
		if len(answers) != 0 {
			sortedAnswers = append(sortedAnswers, answers)
		}

	}

	for i := range qcm.Questions {
		qcm.Questions[i].Circle = sortedQuestions[i]
		for j := range qcm.Questions[i].Answers {
			qcm.Questions[i].Answers[j].Circle = sortedAnswers[i][j]
		}

	}

	/*
		// Sérialiser en JSON
		qcmJSON, err := json.Marshal(qcm)
		if err != nil {
			log.Println("json.Marshal(qcm) error : %v", err)
			return qcm, err
		}
	*/

	return qcm, nil
}
