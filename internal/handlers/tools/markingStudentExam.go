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

func MarkingStudentExam(userID int64, username, tempDir string, exam config.Exam, ctx context.Context, queries *db.Queries) (config.MarkExam, error) {
	var markExam config.MarkExam
	markExam.Status = false

	// re-construire l'exam de base avec typst et faire le png
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

	typstFilePath, ok := TypstWriter(tempDir, username, qcm, config.ExamQCM)
	if !ok {
		log.Println("TypstWriter return not ok")
		return markExam, ErrMarkingStudentExam
	}

	pages, ok := ExportTypstToPNGs(typstFilePath)
	if !ok {
		log.Println("ExportTypstToPNGs return not ok")
		return markExam, ErrMarkingStudentExam
	}
	if len(pages) != len(exam.Pages) || len(pages) != int(datas.PageTot) {
		log.Println("From MarkingStudentExam -> exported page count does not match scanned or database page count")
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
	for pageNumber := 1; pageNumber <= len(pages); pageNumber++ {
		if scannedPages[pageNumber] == "" {
			log.Printf("From MarkingStudentExam -> scanned page %d is missing", pageNumber)
			return markExam, ErrMarkingStudentExam
		}
	}

	var answersState []int
	var homoPages []config.HomoPage
	for i, page := range pages {

		imgName := filepath.Base(page)

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

		// homographie de la page sur le png de typst
		homoName := Homography(tempDir, pageName, imgName)
		if homoName == "" {
			log.Printf("From MarkingStudentExam -> Homography failed for page %d", pagesNumbers[i])
			return markExam, ErrMarkingStudentExam
		}
		homoPages = append(homoPages, config.HomoPage{
			Name:    homoName,
			Content: pageContent,
		})

		// DrawCircleOnQcm(tempDir, homoName, "sur_homo_", pageContent.Questions, pageContent.Answers)

		// sur l'homographie, regarder les réponses eleves

		state, err := GetAnswersState(tempDir, homoName, pageContent.Answers)
		if err != nil {
			return markExam, err
		}

		answersState = append(answersState, state...)

	}

	if err := validateMarkingVectors(qcm, homoPages, answersState); err != nil {
		log.Printf("From MarkingStudentExam -> invalid marking vectors: %v", err)
		return markExam, ErrMarkingStudentExam
	}

	// faire une liste question - reponse et compararer
	questionsState := CountingPoints(qcm, answersState)
	mark, tot := CountingTotalPoint(questionsState)
	skill, themeSkill := GetThemeSkill(qcm, questionsState)

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

		DrawMarking(tempDir, page.Name, questionsMark, page.Content.Questions, answersMark, page.Content.Answers)

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
	filesToRm = append(filesToRm, typstFilePath)
	filesToRm = append(filesToRm, pdfNames...)
	filesToRm = append(filesToRm, pages...)
	for _, p := range exam.Pages {
		filesToRm = append(filesToRm, filepath.Join(tempDir, p.Name))
	}
	if err := RemoveFiles(filesToRm); err != nil {
		log.Println("can't remove files")
	}

	markExam = config.MarkExam{
		Status:     true,
		ExamName:   qcm.Name,
		FirstName:  qcm.Student.FirstName,
		LastName:   qcm.Student.LastName,
		ClassName:  qcm.Student.ClassCodes.Name,
		Pages:      len(pages),
		Score:      mark,
		Total:      tot,
		Skill:      skill,
		ThemeSkill: themeSkill,
	}
	return markExam, nil
}
