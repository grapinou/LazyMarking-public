package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

var ErrMarkingStudentExam = errors.New("can't make markingStudentExam")
var ErrMarkingHistoricalReference = errors.New("historical marking reference failure")

func MarkingStudentExam(userID int64, username, tempDir string, exam config.Exam, ctx context.Context, queries *db.Queries) (config.MarkExam, error) {
	var markExam config.MarkExam
	markExam.Status = false

	datas, err := queries.GetStudentContentExam(ctx, db.GetStudentContentExamParams{
		StudentExamID: exam.StudentExamID,
		UserID:        userID,
	})
	if err != nil {
		log.Printf("From MarkingStudentExam -> GetStudentContentExam return error : %v", err)
		return markExam, ErrMarkingStudentExam
	}

	// unmarchal qcm
	var qcm config.QCM
	if err := json.Unmarshal([]byte(datas.Content), &qcm); err != nil {
		log.Printf("From MarkingStudentExam -> Unmarshal return error : %v", err)
		return markExam, ErrMarkingStudentExam
	}

	markExam.FirstName = qcm.Student.FirstName
	markExam.LastName = qcm.Student.LastName

	// vérification du nombre de pages
	if len(exam.Pages) != int(datas.PageTot) {
		log.Println("From MarkingStudentExam -> exam not treated because pages is missing")
		return markExam, ErrMarkingStudentExam
	}

	// tri des pages par mesure de sécurité
	var pagesNumbers []int
	scannedPages := make(map[int]string, len(exam.Pages))
	for _, n := range exam.Pages {
		pagesNumbers = append(pagesNumbers, n.Number)
		scannedPages[n.Number] = n.Name
	}
	sort.Ints(pagesNumbers)
	for pageNumber := 1; pageNumber <= int(datas.PageTot); pageNumber++ {
		if scannedPages[pageNumber] == "" {
			log.Printf("From MarkingStudentExam -> scanned page %d is missing", pageNumber)
			return markExam, ErrMarkingStudentExam
		}
	}
	pages, referenceCleanup, err := resolveMarkingPageReferences(
		ctx, queries, userID, username, exam.StudentExamID, pagesNumbers,
		func() ([]string, []string, error) {
			return renderLegacyMarkingReferences(tempDir, username, qcm)
		},
	)
	if err != nil {
		log.Printf("From MarkingStudentExam -> resolve page references: %v", err)
		return markExam, fmt.Errorf("%w: %v", ErrMarkingHistoricalReference, err)
	}
	if len(pages) != len(exam.Pages) || len(pages) != int(datas.PageTot) {
		log.Println("From MarkingStudentExam -> reference page count does not match scanned or database page count")
		return markExam, ErrMarkingStudentExam
	}

	var answersState []int
	var answerDetections []config.AnswerDetection
	var homoPages []config.HomoPage
	var stagedAlignedPages []config.StagedAlignedPage
	for i, page := range pages {

		pagedatas, err := queries.GetPageContent(ctx, db.GetPageContentParams{
			StudentExamID: exam.StudentExamID,
			Page:          int64(pagesNumbers[i]),
			UserID:        userID,
		})
		if err != nil {
			log.Printf("From MarkingStudentExam -> GetPageContent return error : %v", err)
			return markExam, ErrMarkingStudentExam
		}

		var pageContent config.PageContent
		if err := json.Unmarshal([]byte(pagedatas), &pageContent); err != nil {
			log.Printf("From MarkingStudentExam -> Unmarshal return error : %v", err)
			return markExam, ErrMarkingStudentExam
		}

		// DrawCircleOnQcm(tempDir, imgName, "sur_png_", pageContent.Questions, pageContent.Answers)

		// on s'assure de prendre la bonne page
		pageName := scannedPages[pagesNumbers[i]]

		// DrawCircleOnQcm(tempDir, pageName, "sur_png_", pageContent.Questions, pageContent.Answers)

		// Homography consumes the validated native pre-QR PNG directly. Legacy
		// pages are the only references rendered into the marking workspace.
		homoName, err := Homography(tempDir, pageName, page)
		if err != nil {
			log.Printf("From MarkingStudentExam -> Homography failed for page %d: %v", pagesNumbers[i], err)
			return markExam, ErrMarkingStudentExam
		}
		homoPages = append(homoPages, config.HomoPage{
			Name:    homoName,
			Content: pageContent,
		})

		// DrawCircleOnQcm(tempDir, homoName, "sur_homo_", pageContent.Questions, pageContent.Answers)

		// sur l'homographie, regarder les réponses eleves

		detections, err := GetAnswerDetections(tempDir, homoName, pageContent.Answers)
		if err != nil {
			return markExam, err
		}

		answerDetections = append(answerDetections, detections...)
		answersState = append(answersState, answerDetectionScoringStates(detections)...)
		stagedPath, err := StageMarkingAlignedPage(tempDir, exam.StudentExamID, pagesNumbers[i], homoName)
		if err != nil {
			log.Printf("From MarkingStudentExam -> stage aligned page %d: %v", pagesNumbers[i], err)
			return markExam, err
		}
		stagedAlignedPages = append(stagedAlignedPages, config.StagedAlignedPage{PageExam: pagesNumbers[i], Path: stagedPath})

	}

	if err := validateMarkingVectors(qcm, homoPages, answersState); err != nil {
		log.Printf("From MarkingStudentExam -> invalid marking vectors: %v", err)
		return markExam, ErrMarkingStudentExam
	}

	// faire une liste question - reponse et compararer
	questionsState := CountingPoints(qcm, answersState)
	mark, tot := CountingTotalPoint(questionsState)
	skill, themeSkill := GetThemeSkill(qcm, questionsState)
	detailedResult, err := BuildMarkingCopyResult(
		exam.StudentExamID,
		int(datas.PageTot),
		len(exam.Pages),
		qcm,
		questionsState,
		answerDetections,
	)
	if err != nil {
		log.Printf("From MarkingStudentExam -> build detailed result: %v", err)
		return markExam, ErrMarkingStudentExam
	}

	var answersQCM []int
	for _, question := range qcm.Questions {
		for _, answer := range question.Answers {
			answersQCM = append(answersQCM, int(answer.State))
		}
	}

	// faire la correction sur les feuilles
	var pdfNames []string
	var filesToRm []string
	for _, page := range homoPages {

		questionsMark := questionsState[:len(page.Content.Questions)]
		questionsState = questionsState[len(page.Content.Questions):]

		answersMark := answersQCM[:len(page.Content.Answers)]
		answersQCM = answersQCM[len(page.Content.Answers):]
		effectiveAnswersMark := answersState[:len(page.Content.Answers)]
		answersState = answersState[len(page.Content.Answers):]

		DrawMarking(tempDir, page.Name, questionsMark, page.Content.Questions, answersMark, effectiveAnswersMark, page.Content.Answers)

		name, err := ConvertPngTopdf(tempDir, page.Name)
		if err != nil {
			return markExam, err
		}
		pdfNames = append(pdfNames, filepath.Join(tempDir, name))
		filesToRm = append(filesToRm, filepath.Join(tempDir, page.Name))
	}

	// faire un pdf
	name := filepath.Join(tempDir, fmt.Sprintf("student-exam-%d.pdf", exam.StudentExamID))
	if err := MergePdf(pdfNames, name); err != nil {
		log.Println("can't merge pdf")
		return markExam, err
	}

	// cleaning
	filesToRm = append(filesToRm, pdfNames...)
	filesToRm = append(filesToRm, referenceCleanup...)
	for _, p := range exam.Pages {
		filesToRm = append(filesToRm, filepath.Join(tempDir, p.Name))
	}
	if err := RemoveFiles(filesToRm); err != nil {
		log.Println("can't remove files")
	}

	markExam = config.MarkExam{
		StudentExamID:  exam.StudentExamID,
		Status:         true,
		ExamName:       qcm.Name,
		FirstName:      qcm.Student.FirstName,
		LastName:       qcm.Student.LastName,
		ClassName:      qcm.Student.ClassCodes.Name,
		Pages:          len(pages),
		Score:          mark,
		Total:          tot,
		Skill:          skill,
		ThemeSkill:     themeSkill,
		DetailedResult: &detailedResult,
		AlignedPages:   stagedAlignedPages,
	}
	return markExam, nil
}
